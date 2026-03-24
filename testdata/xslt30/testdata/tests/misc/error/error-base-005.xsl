<xsl:package 
  name="http://www.w3.org/xslt30tests/error-base-005"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/error-base-005"
  declared-modes="yes"
  exclude-result-prefixes="xs p">
  
  <xsl:mode/>
  <xsl:mode name="m" visibility="public" on-no-match="shallow-copy"/>
  
  <xsl:template match="a">
    <a/>
  </xsl:template> 
  
  <xsl:template match="b">
    <b/>
  </xsl:template>  
  
</xsl:package>   