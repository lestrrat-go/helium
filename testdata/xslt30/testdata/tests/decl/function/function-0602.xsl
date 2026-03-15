<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<?dependency saxon?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:saxon="http://saxon.sf.net/" exclude-result-prefixes="saxon xs">

<!-- Show that by default stylesheet functions override vendor-supplied functions -->
<!-- This test doesn't prove much except when run on Saxon, but it should produce reproducible results
     on all implementations -->

<xsl:function name="saxon:line-number" as="xs:integer">
    <xsl:sequence select="342"/>
</xsl:function>

  <xsl:template match="/">
      <out><xsl:value-of select="saxon:line-number()"/></out>
  </xsl:template>


</xsl:stylesheet>
