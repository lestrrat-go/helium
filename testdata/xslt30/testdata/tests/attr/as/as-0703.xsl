<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.mytest.net"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test type of tunnel xsl:with-param that has @as="element(QName, xs:untyped?)" with: 
				- @select selecting an element node from input file
				- no @select and sequence constructor has xsl:element
				- no @select and sequence constructor has an LRE -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="/doc">
      <out>
         <xslt:apply-templates>
            <xslt:with-param name="par1" select="item-list/item2" as="element(item2, xs:untyped?)"
               tunnel="yes"/>
            <xslt:with-param name="par2" as="element(elem, xs:untyped?)" tunnel="yes">
               <xslt:element name="elem">hello</xslt:element>
            </xslt:with-param>
            <xslt:with-param name="par3" as="element(my:item, xs:untyped?)" tunnel="yes">
               <my:item>
                  <xslt:value-of select="'my element'"/>
               </my:item>
            </xslt:with-param>
         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="item-list">
      <xslt:apply-templates/>
   </xslt:template>

   <xslt:template match="item1">
      <xslt:param name="par1" select="3.5" tunnel="yes"/>
      <xslt:param name="par2" select="3.5" tunnel="yes"/>
      <xslt:param name="par3" select="3.5" tunnel="yes"/>
      <par1>
         <xslt:value-of select="$par1"/>
         <xslt:value-of select="$par1 instance of element(item2, xs:untyped?)"/>
      </par1>
      <par2>
         <xslt:value-of select="$par2"/>
         <xslt:value-of select="$par2 instance of element(elem, xs:untyped?)"/>
      </par2>
      <par3>
         <xslt:value-of select="$par3"/>
         <xslt:value-of select="$par3 instance of element(my:item, xs:untyped?)"/>
      </par3>
   </xslt:template>

   <xslt:template match="item2"> </xslt:template>
</xslt:transform>
