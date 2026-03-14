<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:math="http://www.w3.org/2005/xpath-functions/math" exclude-result-prefixes="xs math"
   version="3.0" xmlns="urn:FOO">
   
   <xsl:variable name="input">
      <a><b/></a>
   </xsl:variable>

   <xsl:include href="xpath-default-namespace-1202inc.xsl"/>
   
   <xsl:template match="b" xpath-default-namespace="urn:FOO">
      <B/>
   </xsl:template>
   
   <xsl:template name="xsl:initial-template">
      <xsl:apply-templates select="$input/*"/>
   </xsl:template>
</xsl:stylesheet>
