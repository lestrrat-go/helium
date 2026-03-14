<?xml version='1.0'?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test base URI of an embedded stylesheet module -->

<xsl:include href="include-0103a.xml#embedded"/>


<xsl:template match="/">
    <a><xsl:call-template name="x"/></a>
</xsl:template>

</xsl:stylesheet>
