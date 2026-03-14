<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: build-tree=no on xsl:result-document.-->
   
   <xsl:output name="f" item-separator="|" omit-xml-declaration="yes"/>

   <xsl:template match="/">
      <xsl:result-document build-tree="no" format="f" item-separator="#absent" method="xml">
         <xsl:comment>begin</xsl:comment>
         <xsl:sequence select="1 to count(//*)"/>
         <xsl:comment>end</xsl:comment>
      </xsl:result-document>
   </xsl:template>
</xsl:transform>
