<?xml version="1.0"?> 

<!-- Test copying of attribute from a temporary tree (docbook bug clearance) -->
<?spec xslt#copy-of?>   
<!-- See bug 529362 -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:variable name="tree">
    <a xmlns="http://some.uri/" z="1"/>
  </xsl:variable>

  <xsl:template match="/">
    <out xmlns="http://some.uri/">
      <xsl:copy-of select="$tree/some:a/@*" xmlns:some="http://some.uri/"/>
    </out>
  </xsl:template>
</xsl:stylesheet>
