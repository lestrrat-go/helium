<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test with xsl:template that has @as="attribute(QName, xs:untypedAtomic)*" 
  				and returns the empty sequence or a sequence of xsl:attribute.-->

   <xslt:template match="/">
      <out>
         <e1>
            <xslt:call-template name="a1"/>
         </e1>
         <e2>
            <xslt:call-template name="a2"/>
         </e2>
      </out>
   </xslt:template>

   <xslt:template name="a1" as="attribute(att1, xs:untypedAtomic)*">
      <xslt:sequence select="()"/>
   </xslt:template>

   <xslt:template name="a2" as="attribute(att, xs:untypedAtomic)*">
      <xslt:attribute name="att">1</xslt:attribute>
      <xslt:attribute name="att">2</xslt:attribute>
      <xslt:attribute name="att">3</xslt:attribute>
      <xslt:attribute name="att">4</xslt:attribute>
   </xslt:template>
</xslt:transform>
