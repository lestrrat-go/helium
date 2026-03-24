<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#constructing-complex-content?>
<?error XTDE0410?>
    <!-- Purpose: Verify adding an attribute to an element after a PI has
       been added to it is an error. The attributes can be ignored.
       The spec doesn't explicitly say this is disallowed, as it does
       for child elements, but it makes sense to have the same treatment. -->

<xsl:template match="doc">
  <Out>
    <xsl:element name="Element1">
      <xsl:attribute name="Att1">OK</xsl:attribute>
      <xsl:processing-instruction name="my-pi">type="text/css"</xsl:processing-instruction>
      <xsl:attribute name="AttX">Wrong</xsl:attribute>
      <xsl:element name="Element2"/>
    </xsl:element>	  
  </Out>
</xsl:template>

</xsl:stylesheet>