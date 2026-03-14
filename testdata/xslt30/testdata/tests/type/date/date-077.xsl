<?xml version="1.0" encoding="UTF-8" standalone="no"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
 version="2.0" xmlns:xs="http://www.w3.org/2001/XMLSchema">
 
 <?spec fo#casting-to-datetimes?>
 
<xsl:template match="/">
<x>
<a><xsl:value-of select="xs:date('2004-02-02') cast as xs:dateTime"/></a>
<b><xsl:value-of select="xs:date('2004-02-02+05:00') cast as xs:dateTime"/></b>
</x>
</xsl:template>

</xsl:stylesheet>
