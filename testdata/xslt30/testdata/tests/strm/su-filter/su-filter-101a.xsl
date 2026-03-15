<xsl:package version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  name="http://su-filter-101a/"
  package-version="1.0"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/functions" exclude-result-prefixes="map xs err f">
  
  <xsl:variable name="cat" select="'M'" visibility="public"/>
  
  <xsl:function name="f:filter-001" as="node()?" streamability="filter" visibility="public">
    <xsl:param name="input" as="node()"/>
    <xsl:sequence select="$input[contains(@CAT, $cat)]"/>
  </xsl:function>
 

</xsl:package>
