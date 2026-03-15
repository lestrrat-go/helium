<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test with tunnel xsl:with-param without @select, where the sequence constructor contains an 
  				xs:untypedAtomic obtained at run time and @as=xs:double.   -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="/">
      <out>
         <xslt:apply-templates>
            <xslt:with-param name="par1" as="xs:double" tunnel="yes">
               <item>
                  <xslt:value-of select="doc//item1"/>
               </item>
            </xslt:with-param>
         </xslt:apply-templates>
      </out>
   </xslt:template>

   <xslt:template match="item-list">
      <xslt:apply-templates/>
   </xslt:template>

   <xslt:template match="item1">
      <xslt:param name="par1" select="3.5" tunnel="yes"/>
      <par1>
         <xslt:value-of select="$par1"/>
         <xslt:value-of select="$par1 instance of xs:double"/>
      </par1>
   </xslt:template>

   <xslt:template match="item2"> </xslt:template>
</xslt:transform>
