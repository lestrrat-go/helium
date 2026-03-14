<?xml version="1.0"?>
<?spec xslt#creating-processing-instructions?>
<!-- test that multiple ?> delimiters can be adjusted in the output -->
<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:a="http://aa.com/" xmlns:b="http://bb.com/">
   <xsl:template match="/">
      <a><xsl:processing-instruction name="pidl" select="'----?>?>?>?>?>----?>'"/></a>
   </xsl:template>
</xsl:stylesheet>
