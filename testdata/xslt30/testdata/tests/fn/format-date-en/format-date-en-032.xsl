<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: fallback to English for unsupported language.
       Test assumes (a) that Iberian (xib) is not supported, and (b) that the fallback language is English.
       Changed from lang="fo" to lang="xib": See bug 862. -->

  <xsl:param name="b" as="xs:date" select="xs:date('2006-03-01')"/>

  <xsl:template name="main">
<out>;
<z><xsl:value-of select="format-date($b, '[MNn]', 'xib', (), ())"/></z>;
</out>     
  </xsl:template>

</xsl:stylesheet>