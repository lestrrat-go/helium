<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
    <!-- Purpose: String value of the root node is the concatenation of the string
       values of all text node descendants of the root node in document order. -->

<xsl:template match="/">
<root>
	<xsl:value-of select="."/>
</root>
</xsl:template>

</xsl:stylesheet>