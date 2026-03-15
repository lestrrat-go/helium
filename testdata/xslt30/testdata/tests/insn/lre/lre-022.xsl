<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
<?error  XTDE0820?>
    <!-- Purpose: Test error if required element name is null.-->
  <!-- Test modified by MHK to make it a dynamic error -->

<xsl:template match="doc">
  <xsl:param name="n" select="''"/>
  <out>
    <xsl:element name="{$n}">
      This should be directly inside the out element.
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>
