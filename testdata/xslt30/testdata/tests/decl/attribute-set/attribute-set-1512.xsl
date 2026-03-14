<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test inheritance of attribute sets. A literal result element
                is referring an attribute set that is defined by two separate
                <xsl:attribute-set.../> elements with the same name. Both
                these elements have a use-attribute-sets attribute, which
                means that we have a single attribute set that inherits from
                two other attribute sets. Both parents attribute sets have
                attributes that are overridden by the child.-->

  <xsl:template match="/">
    <out xsl:use-attribute-sets="child">
      <xsl:attribute name="location">Wonderland</xsl:attribute>
    </out>
  </xsl:template>

  <xsl:attribute-set name="child" use-attribute-sets="alice">
    <xsl:attribute name="follow">yellowbrickroad</xsl:attribute>
    <xsl:attribute name="rabbithole">shallow</xsl:attribute>
  </xsl:attribute-set>

  <xsl:attribute-set name="child" use-attribute-sets="rabbit">
    <xsl:attribute name="follow">theleader</xsl:attribute>
    <xsl:attribute name="alice">intoxicated</xsl:attribute>
  </xsl:attribute-set>

  <xsl:attribute-set name="rabbit">
    <xsl:attribute name="rabbithole">deep</xsl:attribute>
  </xsl:attribute-set>

  <xsl:attribute-set name="alice">
    <xsl:attribute name="alice">ondrugs</xsl:attribute>
  </xsl:attribute-set>

</xsl:stylesheet>
