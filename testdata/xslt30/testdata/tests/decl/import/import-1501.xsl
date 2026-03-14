<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Purpose: Test of import (down and up directory tree) using ..\main_import.xsl
   from two separate subdiretories. Filename deliberately repeated! -->

<xsl:import href="fragments/frag-subdir/main_import.xsl"/>

<xsl:template match="test">
<out>
From Stylesheet: <xsl:value-of select="."/>
<xsl:apply-imports/>
</out>
</xsl:template>

</xsl:stylesheet>