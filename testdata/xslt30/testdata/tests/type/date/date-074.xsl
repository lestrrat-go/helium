<?xml version="1.0" encoding="UTF-8" standalone="no"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
 version="2.0" xmlns:xs="http://www.w3.org/2001/XMLSchema">
 
  <?spec fo#casting-to-datetimes?>
 
 <xsl:template match="/">
<a><xsl:value-of select="xs:date('-0055-01-12')"/></a>
</xsl:template>

</xsl:stylesheet>
