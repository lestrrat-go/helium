<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0">

<xsl:template match="doc">
	<xsl:element name="dummy" att="error" /> <!-- error: @att is no allowed on an xsl:element -->
</xsl:template>

</xsl:stylesheet>