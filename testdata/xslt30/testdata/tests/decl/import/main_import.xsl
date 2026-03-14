<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!--   This is the last file in the import chain. Main XSLT file is in same directory. -->

<xsl:template match="test">
In ImpIncl: <xsl:value-of select="."/>
</xsl:template>

</xsl:stylesheet>