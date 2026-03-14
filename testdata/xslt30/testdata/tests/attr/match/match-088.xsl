<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">


  <!-- Test of patterns with kind test document-node() -->
  
<xsl:template match="text()"/>

<xsl:template match="document-node()/child::element()">
  <elem>
    <xsl:value-of select="name()"/>
  </elem>
</xsl:template >  

<xsl:template match="document-node()">
	<out>
	<xsl:text>Document_Node_Found</xsl:text>
	<xsl:apply-templates />
	</out>
</xsl:template>

</xsl:stylesheet>
