<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- PURPOSE:  Test xsl:evaluate() with variables -->
<!-- Formerly test saxon070 -->



<xsl:template match="/">
<out>
  <xsl:evaluate xpath="add" with-params="map{xs:QName('p1'): 6}">
    <xsl:with-param name="p1" as="xs:integer" select="6"/>
    <xsl:with-param name="p2" as="xs:integer" select="(1,'banana',3,4)[year-from-date(current-date()) idiv 1000]"/>
  </xsl:evaluate> 
</out>
</xsl:template>
</xsl:stylesheet>