<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-time(): timezones in +nn:nn notation -->

  <xsl:param name="t" as="xs:dateTime" select="xs:dateTime('0985-03-01T09:15:06.456Z')"/>  

<xsl:template name="main">
<out>;
<xsl:for-each select="-28 to +28">
<x><xsl:value-of select="format-dateTime(adjust-dateTime-to-timezone(
                            $t, .*xs:dayTimeDuration('PT30M')), '[h01][m01][Z]')"/></x>;
</xsl:for-each>                            
</out>
</xsl:template> 


</xsl:stylesheet>