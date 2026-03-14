<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  version="3.0" exclude-result-prefixes="xs map">

  <xsl:variable name="f" as="function(*)" static="yes" select="xs:date#1"/>
  
  <xsl:variable name="g" as="xs:date" static="yes" select="$f('2017-10-10')"/>

  <xsl:template name="xsl:initial-template">
    <out><xsl:value-of select="format-date($g, '[D] [Mi] [Y0001]')"/></out>
  </xsl:template>



</xsl:stylesheet>
