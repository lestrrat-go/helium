<?xml version="1.0" encoding="UTF-8" standalone="no"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
 version="2.0" xmlns:xs="http://www.w3.org/2001/XMLSchema">
 
 <?spec fo#casting-to-datetimes?>
 
<xsl:template match="/">
<x>
<a><xsl:value-of select="xs:gYear('195778')"/></a>
<b><xsl:value-of select="xs:gYearMonth('195778-06')"/></b>
</x>
</xsl:template>

</xsl:stylesheet>
