<?xml version="1.0" encoding="UTF-8" standalone="no"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
 version="2.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
 
  <?spec fo#casting-to-durations?>
 
 <xsl:template match="/">
<a><xsl:value-of select="xs:dayTimeDuration('P1234567890DT1234567890H1234567890M1234567890.125S')"/></a>
</xsl:template>

</xsl:stylesheet>
