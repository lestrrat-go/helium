<?xml version="1.0"?>
<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:a="http://aa.com/" xmlns:b="http://bb.com/">
<?spec xslt#xsl-element?>
   <xsl:template match="/">
      <xsl:element name="{'b:blah'}"/>
   </xsl:template>
</xsl:stylesheet>
