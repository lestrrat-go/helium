<?xml version="1.0"?>
<?spec xslt#constructing-sequences?>
<!-- initial comment -->
<?pi-start?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- PURPOSE:   Test xsl:sequence adding a document node to a result tree -->
<xsl:template match="doc">
    <zzz>
        <xsl:text>{{{{{</xsl:text>
        <xsl:sequence select="document('')"/>
        <xsl:text>}}}}}</xsl:text>
    </zzz>
</xsl:template>
</xsl:stylesheet>
<?pi-end?>
<!-- final comment -->