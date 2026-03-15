<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:my="http://uri.test" xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="my xs" version="2.0">
   <!-- Purpose: Test with xsl:template that has @as="element(*, xs:untyped)+" 
  				and returns a sequence of xsl:element or LREs.-->

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

   <xslt:template name="a1" as="element(*, xs:untyped)+">
      <xslt:element name="elem">hi1</xslt:element>
      <xslt:element name="elem">hi2</xslt:element>
      <xslt:element name="item">hi3</xslt:element>
   </xslt:template>

   <xslt:template name="a2" as="element(*, xs:untyped)+">
      <my:item>1</my:item>
      <item>2</item>
      <my:item>3</my:item>
      <elem>hello</elem>
   </xslt:template>
</xslt:transform>
