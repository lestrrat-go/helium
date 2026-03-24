<xsl:stylesheet
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:bogus="http://bogus">

<?spec xpath#static_context?>
    <!-- Purpose: Test that default namespaces do not apply directly to attributes. -->

<xsl:template match="*[@up]">
  <foo/>
</xsl:template>

<xsl:template match="*[@bogus:up]">
  <!-- this template should not trigger because the element is in the "bogus" namespace,
       but its "up" attribute isn't -->
  <ERROR/>
</xsl:template>

</xsl:stylesheet>

