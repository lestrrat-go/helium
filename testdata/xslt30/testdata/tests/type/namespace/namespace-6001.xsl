<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:ped="http://www.ped.com">

<?spec xslt#xsl-element?>
    <!-- Purpose: Use xsl:element with namespace in AVT, comes out as null string. -->

<xsl:template match="/">
 <out>
   <xsl:element name="{docs/b}" namespace="{docs/b/@href}"/>
 </out>
</xsl:template>

</xsl:stylesheet>
