<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/functions" exclude-result-prefixes="map xs err f">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:strip-space elements="*"/>


  
  <!-- Test arity=1 recursive ascent function returning an ancestor node-->
  
  <xsl:function name="f:outermost-section" as="element(section)" streamability="ascent">
    <xsl:param name="input" as="node()"/>
    <xsl:sequence select="if ($input/ancestor::section) then f:outermost-section($input/ancestor::section[1]) else $input"/>
  </xsl:function>
  
  <xsl:template name="t-005" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/recursive.xml">
      <out>
        <xsl:value-of select="//foot ! f:outermost-section(.) ! @id"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test arity=1 ascent function returning an ancestor node-->
  
  <xsl:function name="f:outermost-section-id"  streamability="ascent">
    <xsl:param name="input" as="node()?"/>
    <xsl:sequence select="if ($input) then if (matches($input/@id, '^[0-9]+\.[0-9]+$')) then $input else f:outermost-section-id($input/..) else ()"/>
  </xsl:function>
  
  <xsl:template name="t-006" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/recursive.xml">
      <out>
        <xsl:value-of select="//foot ! f:outermost-section-id(.) ! @id"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
 

</xsl:stylesheet>
