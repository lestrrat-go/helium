<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
		version="2.0">

<!-- test namespace-uri-from-QName with a null namespace URI -->
   <?spec fo#func-QName?>
   <?spec fo#func-namespace-uri-from-QName?>

<xsl:template match="/">
  <x>
     <xsl:variable name="x" select="namespace-uri-from-QName(QName('', 'testlocal'))"/>
     <xsl:value-of select="count($x), string-length($x)"/>
  </x>
</xsl:template>

</xsl:stylesheet>
