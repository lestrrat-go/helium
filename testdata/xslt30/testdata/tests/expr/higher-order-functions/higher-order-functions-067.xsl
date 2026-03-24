<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:schold="http://www.ascc.net/xml/schematron" xmlns:iso="http://purl.oclc.org/dsdl/schematron"
  xmlns:xhtml="http://www.w3.org/1999/xhtml" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:my="http://eXolutio.com/my/functional"
  xmlns:myin="http://eXolutio.com/my/functional/internal"
  xmlns:oclDate="http://eXolutio.com/my/types/date"
  xmlns:oclString="http://eXolutio.com/my/types/string" 
  version="3.0"
  exclude-result-prefixes="schold xhtml my myin oclDate oclString">
  
  <!--Implementers: please note that overriding process-prolog or process-root is
    the preferred method for meta-stylesheets to use where possible. -->
  <xsl:param name="archiveDirParameter"/>
  <xsl:param name="archiveNameParameter"/>
  <xsl:param name="fileNameParameter"/>
  <xsl:param name="fileDirParameter"/>
  <xsl:variable name="document-uri">
    <xsl:value-of select="document-uri(/)"/>
  </xsl:variable>

  <!-- guarding function preventing dynamic errors --> 
  <xsl:function name="my:checked" as="item()*">
    <xsl:param name="func" as="function() as item()*" />
    <xsl:try select="$func()">
      <xsl:catch>        
        <xsl:sequence select="false()" />
      </xsl:catch>
    </xsl:try>
  </xsl:function>

  <!-- simple HOF, universal quantifier -->
  <xsl:function name="my:forAll" as="xs:boolean">
    <xsl:param name="collection" as="item()*"/>
    <xsl:param name="body" as="function(item()) as xs:boolean"/>
    <xsl:sequence select="every $it in $collection satisfies $body($it) "/>
  </xsl:function>
  
  <xsl:template match="/tournament" priority="1000" mode="M1">
    <xsl:variable name="self" as="item()" select="current()"/>
    <svrl:fired-rule xmlns:svrl="http://purl.oclc.org/dsdl/svrl" context="/tournament"/>    
    
    <xsl:choose>      
      <xsl:when
        test="my:checked(function() { my:forAll($self/matches/day/match, function($m) { $m/start eq $self/start} ) } ) ">
        <xsl:text>OK</xsl:text>  
      </xsl:when>      
      <xsl:otherwise>
        <xsl:text>VIOLATE</xsl:text>  
      </xsl:otherwise>
    </xsl:choose>
    <xsl:apply-templates select="*|comment()|processing-instruction()" mode="M1"/>
  </xsl:template>


  <!--PHASES-->


  <!--PROLOG-->
  <xsl:output xmlns:svrl="http://purl.oclc.org/dsdl/svrl" method="xml" omit-xml-declaration="no"
    standalone="yes" indent="no"/>

  <!--XSD TYPES FOR XSLT2-->


  <!--KEYS AND FUNCTIONS-->


  <!--DEFAULT RULES-->


  <!--MODE: SCHEMATRON-SELECT-FULL-PATH-->
  <!--This mode can be used to generate an ugly though full XPath for locators-->
  <xsl:template match="*" mode="schematron-select-full-path">
    <xsl:apply-templates select="." mode="schematron-get-full-path"/>
  </xsl:template>

  <!--MODE: SCHEMATRON-FULL-PATH-->
  <!--This mode can be used to generate an ugly though full XPath for locators-->
  <xsl:template match="*" mode="schematron-get-full-path">
    <xsl:apply-templates select="parent::*" mode="schematron-get-full-path"/>
    <xsl:text>/</xsl:text>
    <xsl:choose>
      <xsl:when test="namespace-uri()=''">
        <xsl:value-of select="name()"/>
        <xsl:variable name="p_1" select="1+    count(preceding-sibling::*[name()=name(current())])"/>
        <xsl:if test="$p_1&gt;1 or following-sibling::*[name()=name(current())]">[<xsl:value-of
            select="$p_1"/>]</xsl:if>
      </xsl:when>
      <xsl:otherwise>
        <xsl:text>*[local-name()='</xsl:text>
        <xsl:value-of select="local-name()"/>
        <xsl:text>']</xsl:text>
        <xsl:variable name="p_2"
          select="1+   count(preceding-sibling::*[local-name()=local-name(current())])"/>
        <xsl:if test="$p_2&gt;1 or following-sibling::*[local-name()=local-name(current())]"
            >[<xsl:value-of select="$p_2"/>]</xsl:if>
      </xsl:otherwise>
    </xsl:choose>
  </xsl:template>
  <xsl:template match="@*" mode="schematron-get-full-path">
    <xsl:text>/</xsl:text>
    <xsl:choose>
      <xsl:when test="namespace-uri()=''">@<xsl:value-of select="name()"/></xsl:when>
      <xsl:otherwise>
        <xsl:text>@*[local-name()='</xsl:text>
        <xsl:value-of select="local-name()"/>
        <xsl:text>' and namespace-uri()='</xsl:text>
        <xsl:value-of select="namespace-uri()"/>
        <xsl:text>']</xsl:text>
      </xsl:otherwise>
    </xsl:choose>
  </xsl:template>

  <!--MODE: SCHEMATRON-FULL-PATH-2-->
  <!--This mode can be used to generate prefixed XPath for humans-->
  <xsl:template match="node() | @*" mode="schematron-get-full-path-2">
    <xsl:for-each select="ancestor-or-self::*">
      <xsl:text>/</xsl:text>
      <xsl:value-of select="name(.)"/>
      <xsl:if test="preceding-sibling::*[name(.)=name(current())]">
        <xsl:text>[</xsl:text>
        <xsl:value-of select="count(preceding-sibling::*[name(.)=name(current())])+1"/>
        <xsl:text>]</xsl:text>
      </xsl:if>
    </xsl:for-each>
    <xsl:if test="not(self::*)"><xsl:text/>/@<xsl:value-of select="name(.)"/></xsl:if>
  </xsl:template>
  <!--MODE: SCHEMATRON-FULL-PATH-3-->
  <!--This mode can be used to generate prefixed XPath for humans 
	(Top-level element has index)-->
  <xsl:template match="node() | @*" mode="schematron-get-full-path-3">
    <xsl:for-each select="ancestor-or-self::*">
      <xsl:text>/</xsl:text>
      <xsl:value-of select="name(.)"/>
      <xsl:if test="parent::*">
        <xsl:text>[</xsl:text>
        <xsl:value-of select="count(preceding-sibling::*[name(.)=name(current())])+1"/>
        <xsl:text>]</xsl:text>
      </xsl:if>
    </xsl:for-each>
    <xsl:if test="not(self::*)"><xsl:text/>/@<xsl:value-of select="name(.)"/></xsl:if>
  </xsl:template>

  <!--MODE: GENERATE-ID-FROM-PATH -->
  <xsl:template match="/" mode="generate-id-from-path"/>
  <xsl:template match="text()" mode="generate-id-from-path">
    <xsl:apply-templates select="parent::*" mode="generate-id-from-path"/>
    <xsl:value-of select="concat('.text-', 1+count(preceding-sibling::text()), '-')"/>
  </xsl:template>
  <xsl:template match="comment()" mode="generate-id-from-path">
    <xsl:apply-templates select="parent::*" mode="generate-id-from-path"/>
    <xsl:value-of select="concat('.comment-', 1+count(preceding-sibling::comment()), '-')"/>
  </xsl:template>
  <xsl:template match="processing-instruction()" mode="generate-id-from-path">
    <xsl:apply-templates select="parent::*" mode="generate-id-from-path"/>
    <xsl:value-of
      select="concat('.processing-instruction-', 1+count(preceding-sibling::processing-instruction()), '-')"
    />
  </xsl:template>
  <xsl:template match="@*" mode="generate-id-from-path">
    <xsl:apply-templates select="parent::*" mode="generate-id-from-path"/>
    <xsl:value-of select="concat('.@', name())"/>
  </xsl:template>
  <xsl:template match="*" mode="generate-id-from-path" priority="-0.5">
    <xsl:apply-templates select="parent::*" mode="generate-id-from-path"/>
    <xsl:text>.</xsl:text>
    <xsl:value-of
      select="concat('.',name(),'-',1+count(preceding-sibling::*[name()=name(current())]),'-')"/>
  </xsl:template>

  <!--MODE: GENERATE-ID-2 -->
  <xsl:template match="/" mode="generate-id-2">U</xsl:template>
  <xsl:template match="*" mode="generate-id-2" priority="2">
    <xsl:text>U</xsl:text>
    <xsl:number level="multiple" count="*"/>
  </xsl:template>
  <xsl:template match="node()" mode="generate-id-2">
    <xsl:text>U.</xsl:text>
    <xsl:number level="multiple" count="*"/>
    <xsl:text>n</xsl:text>
    <xsl:number count="node()"/>
  </xsl:template>
  <xsl:template match="@*" mode="generate-id-2">
    <xsl:text>U.</xsl:text>
    <xsl:number level="multiple" count="*"/>
    <xsl:text>_</xsl:text>
    <xsl:value-of select="string-length(local-name(.))"/>
    <xsl:text>_</xsl:text>
    <xsl:value-of select="translate(name(),':','.')"/>
  </xsl:template>
  <!--Strip characters-->
  <xsl:template match="text()" priority="-1"/>

  <!--SCHEMA SETUP-->
  <xsl:template match="/">
    <svrl:schematron-output xmlns:svrl="http://purl.oclc.org/dsdl/svrl" title="" schemaVersion="">
      <svrl:active-pattern>
        <xsl:attribute name="document">
          <xsl:value-of select="tokenize(document-uri(/), '/')[last()]"/>
        </xsl:attribute>
        <xsl:attribute name="id">er</xsl:attribute>
        <xsl:attribute name="name">er</xsl:attribute>
        <xsl:apply-templates/>
      </svrl:active-pattern>
      <xsl:apply-templates select="/" mode="M0"/>
      <svrl:active-pattern>
        <xsl:attribute name="document">
          <xsl:value-of select="tokenize(document-uri(/), '/')[last()]"/>
        </xsl:attribute>
        <xsl:attribute name="id">collections</xsl:attribute>
        <xsl:attribute name="name">collections</xsl:attribute>
        <xsl:apply-templates/>
      </svrl:active-pattern>
      <xsl:apply-templates select="/" mode="M1"/>
    </svrl:schematron-output>
  </xsl:template>

  <!--SCHEMATRON PATTERNS-->


  <!--PATTERN er-->


  <!--RULE -->
  <xsl:template match="/tournament" priority="1000" mode="M0">
    <xsl:variable name="self" as="item()" select="current()"/>
    <svrl:fired-rule xmlns:svrl="http://purl.oclc.org/dsdl/svrl" context="/tournament"/>

    <!--ASSERT -->
    <!--<xsl:choose>
      <xsl:when test="my:checked(function() { 1 div 0 })"/>
      <xsl:otherwise>
        <svrl:failed-assert xmlns:svrl="http://purl.oclc.org/dsdl/svrl" test="1 div 0">
          <xsl:attribute name="location">
            <xsl:apply-templates select="." mode="schematron-select-full-path"/>
          </xsl:attribute>
          <svrl:text/>
        </svrl:failed-assert>
      </xsl:otherwise>
    </xsl:choose>-->
    <xsl:apply-templates select="*|comment()|processing-instruction()" mode="M0"/>
  </xsl:template>
  <xsl:template match="text()" priority="-1" mode="M0"/>
  <xsl:template match="@*|node()" priority="-2" mode="M0">
    <xsl:apply-templates select="*|comment()|processing-instruction()" mode="M0"/>
  </xsl:template>

  <!--PATTERN collections-->



  <!--RULE -->
  
 
  <xsl:template match="text()" priority="-1" mode="M1"/>
  <xsl:template match="@*|node()" priority="-2" mode="M1">
    <xsl:apply-templates select="*|comment()|processing-instruction()" mode="M1"/>
  </xsl:template>
</xsl:stylesheet>
