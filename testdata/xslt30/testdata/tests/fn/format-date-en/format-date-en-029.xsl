<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: English AD/BC -->

  <xsl:param name="d1" as="xs:date" select="xs:date('1990-12-01')"/>
  <xsl:param name="d2" as="xs:date" select="xs:date('-0055-12-01')"/>  

  <xsl:template name="main">
<out>;
<z><xsl:value-of select="format-date($d1, '[Y][EN]', 'en', (), ())"/></z>;
<z><xsl:value-of select="format-date($d2, '[Y][EN]', 'en', (), ())"/></z>;   
</out>     
  </xsl:template>

</xsl:stylesheet>