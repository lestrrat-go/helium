<?xml version="1.0"?> 
<?spec xslt#named-templates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" xmlns:foo="http://foo.com">

  <!-- DOCUMENT: http://www.w3.org/TR/xslt -->
  <!-- SECTION: 6 Named Templates -->
  <!-- PURPOSE: Make sure qualified names work for named templates. -->
  <xsl:template match="doc">
    <out>
      <xsl:call-template name="foo:a"/>
    </out>
  </xsl:template>
  
  <xsl:template name="foo:a">
    foo:a
  </xsl:template>

  <xsl:template name="a">
    a
  </xsl:template>
 
</xsl:stylesheet>
