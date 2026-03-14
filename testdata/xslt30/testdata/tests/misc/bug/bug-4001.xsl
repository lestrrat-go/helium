<?xml version="1.0" ?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">

<?spec ser#xml?>
<!-- BUG 5.5/tba -->
<!-- DESCRIPTION: Escaping of ] ] > works incorrectly -->

<xsl:output method="xml"/>
<xsl:template match="/">
    <xsl:copy-of select="."/>
</xsl:template>

</xsl:stylesheet >

