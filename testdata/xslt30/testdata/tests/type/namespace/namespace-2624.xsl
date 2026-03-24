<?xml version="1.0"?>
<xsl:stylesheet
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
<?error ?>
    <!-- Purpose: Test error when xmlns prefix is abused -->
  
<xsl:param name="n" select="'xmlns'"/>  

<xsl:template match = "data">
  <xsl:param name="prefix" select="'a'"/>
  <out>
    <xsl:namespace name="{$n}">http://ns.example.com/</xsl:namespace>
  </out>
</xsl:template>

</xsl:stylesheet>