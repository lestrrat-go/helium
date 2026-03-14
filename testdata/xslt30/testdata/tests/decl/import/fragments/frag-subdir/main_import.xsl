<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!--   This is the second file in the import chain.  -->

<xsl:import href="../main_import.xsl"/>

<xsl:template match="test">
In Frag-Subdir: <xsl:value-of select="."/>
<xsl:apply-imports/>
</xsl:template>

</xsl:stylesheet>