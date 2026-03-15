<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/functions" exclude-result-prefixes="map xs err f">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:strip-space elements="*"/>


  <!-- Test arity=1 filter function -->

  <xsl:function name="f:filter-001" as="node()" streamability="filter" use-when="$RUN">
    <xsl:param name="input" as="node()"/>
    <xsl:sequence select="$input[parent::BOOKS]"/>
  </xsl:function>

  <xsl:template name="t-001" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM) ! f:filter-001(.) / @CAT"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:template name="t-002" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM) ! f:filter-001(.) / WEIGHT"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Test arity=2 filter function -->
  
  <xsl:function name="f:filter-003" as="node()?" streamability="filter" use-when="$RUN">
    <xsl:param name="input" as="node()"/>
    <xsl:param name="test" as="xs:boolean"/>
    <xsl:sequence select="$input[$test]"/>
  </xsl:function>
  
  <xsl:template name="t-003" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM) ! f:filter-003(., true()) / @CAT"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:template name="t-004" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM) ! f:filter-003(., false()) / @CAT"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test arity=2 filter function -->

  <xsl:function name="f:get-attribute-value-003" as="xs:string"  streamability="filter" use-when="false() and $RUN">
    <xsl:param name="element" as="node()"/>
    <xsl:param name="attribute-name" as="xs:string"/>
    <xsl:sequence select="string($element/@*[local-name() = $attribute-name])"/>
  </xsl:function>

  <xsl:template name="t-005" use-when="false() and $RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="/BOOKLIST/BOOKS/ITEM/WEIGHT ! f:get-attribute-value-003(., 'UNIT')"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test arity=2 filter function with recursion -->
  
  <xsl:function name="f:get-inherited-attribute-value-004"  streamability="filter" as="xs:string?" use-when="false() and $RUN">
    <xsl:param name="element" as="node()?"/>
    <xsl:param name="attribute-name" as="xs:string"/>
    <xsl:sequence select="
      if (empty($element)) then ()
      else if (exists($element/@*[local-name() = $attribute-name])) 
      then string($element/@*[local-name() = $attribute-name])
      else f:get-inherited-attribute-value-004($element/.., $attribute-name)"/>
  </xsl:function>
  
  <xsl:template name="t-006" use-when="false() and $RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:sequence select="/BOOKLIST/BOOKS/ITEM/WEIGHT ! f:get-inherited-attribute-value-004(., 'CAT')"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
 

</xsl:stylesheet>
