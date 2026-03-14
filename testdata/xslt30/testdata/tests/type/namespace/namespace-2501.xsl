<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#extension-attributes?>
  <!-- PURPOSE:  Testing "An element from the XSLT namespace may 
                 have any attribute not from the XSLT namespace, provided 
                 that the expanded name of the attribute has a non-null 
                 namespace URI." -->
  <xsl:template match="doc">
    <out xmlns:foo="http://foo.com">
      <xsl:copy-of select="doc2" foo:test="0"/>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
