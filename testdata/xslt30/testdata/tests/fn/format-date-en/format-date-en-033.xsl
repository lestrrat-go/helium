<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: fallback to English for unsupported calendar.
       Test assumes that Cooch Behar Era calendar is not supported -->

  <xsl:param name="b" as="xs:date" select="xs:date('2006-03-01')"/>

  <xsl:template name="main">
<out>;
<z><xsl:value-of select="format-date($b, '[M01]', 'en', 'CB', ())"/></z>;
</out>     
  </xsl:template>

</xsl:stylesheet>