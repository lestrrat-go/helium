<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:ped="http://www.ped.com">

<?spec xslt#xsl-element?>
    <!-- Purpose: Use xsl:element with namespace that is AVT, URI matches one already in scope. -->
  <!-- Requested name has no NS prefix, so we probably get just that. -->

<xsl:template match="/">
 <out>
   <xsl:element name="{docs/a}" namespace="{docs/a/@href}"/>
 </out>
</xsl:template>

</xsl:stylesheet>
