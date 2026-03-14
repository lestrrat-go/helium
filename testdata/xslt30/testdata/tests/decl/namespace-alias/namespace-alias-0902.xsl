<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:X="XSL" exclude-result-prefixes="xs" version="3.0">
  <xsl:namespace-alias stylesheet-prefix="X" result-prefix="xsl"/>
  <xsl:template name="main">
    <xsl:variable name="properties" as="item()*">
      <X:attribute/>
    </xsl:variable>
    <result>
      <xsl:sequence select="$properties"/>
    </result>
  </xsl:template>
</xsl:stylesheet>