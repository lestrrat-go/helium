<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#attribute-value-templates?>
  <!-- Purpose: Testing generation of null attribute.
     Bug: if $From was null, it was not outputting the attribute. -->
  <!-- Author: Geoff Crowther -->

<xsl:param name="From"/>

<xsl:template match="/">
  <out>
    <setvar name="From" value="{$From}"/>
  </out>
</xsl:template>

</xsl:stylesheet>
