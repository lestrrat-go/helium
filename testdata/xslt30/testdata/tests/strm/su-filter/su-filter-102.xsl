<xsl:package version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  name="http://su-filter-101/"
  package-version="1.0"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/functions" exclude-result-prefixes="map xs err f">
  
  <xsl:use-package name="http://su-filter-101a/" package-version="1.0">
    <xsl:override>
      <xsl:function name="f:filter-001" as="node()?" streamability="filter" visibility="public">
        <xsl:param name="input" as="node()"/>
        <xsl:sequence select="$input[not(contains(@CAT, $cat))]"/>
      </xsl:function>
    </xsl:override>
  </xsl:use-package>
  
  <xsl:template name="xsl:initial-template" visibility="public">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM) ! f:filter-001(.) / TITLE"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  
</xsl:package>
