<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
<?error ?>
    <!-- Purpose: Test to make sure xsl:attribute isn't added to the containing 
       element when an xsl:element is ignored. Should see a warning that xyz:foo 
       was not created.	The <yyy/> element is placed directly within out, but the 
       attribute isn't. -->

<xsl:template match = "/">
  <xsl:param name="prefix" select="'xyz'"/>
  <out>
    <xsl:element name="{$prefix}:foo">
      <xsl:attribute name="baz"/>
      <yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>


