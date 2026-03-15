<?xml version="1.0"?>
<?spec fo#func-base-uri?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <!-- Test that a processing instruction may have a base URI distinct from its parent element -->

<xsl:template match="test">
  <out>
    <e><xsl:value-of select="substring-after(base-uri(*), '/fn/')"/></e>
    <p><xsl:value-of select="substring-after(base-uri(processing-instruction()), '/fn/')"/></p>
  </out>
</xsl:template>

</xsl:stylesheet>
