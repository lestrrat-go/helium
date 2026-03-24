<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:f="http://local-functions/"
  version="3.0" exclude-result-prefixes="xs map f">

  <xsl:variable name="f" as="function(*)" static="no" select="function($in){xs:date(substring($in, 7, 4) || '-' || substring($in, 1, 2) || '-' ||substring($in, 4, 2))}"/>
  
  <xsl:param name="g" as="xs:date" select="$f('04-18-1924')"/>

  <xsl:template name="xsl:initial-template">
    <out><xsl:value-of select="format-date($g, '[D] [Mi] [Y0001]')"/></out>
  </xsl:template>
  

</xsl:stylesheet>
