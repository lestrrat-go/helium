<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test of global xsl:param with a sequence constructor of 
  				LREs and @as= built-in derived atomic type. Value of LRE is 
  				in the lexical space of the type in @as. Verify the param 
  				is of the type specified in @as. -->

   <xslt:param name="var1" as="xs:normalizedString">
      <item>like</item>
   </xslt:param>

   <xslt:param name="var2" as="xs:token">
      <item>two organizations</item>
   </xslt:param>

   <xslt:param name="var3" as="xs:language">
      <item>PT</item>
   </xslt:param>

   <xslt:param name="var4" as="xs:Name">
      <item>a:_underscore_</item>
   </xslt:param>

   <xslt:param name="var5" as="xs:NCName">
      <item>vwxyz12</item>
   </xslt:param>

   <xslt:param name="var6" as="xs:NMTOKEN">
      <item>business-enforcement::link-chain.common</item>
   </xslt:param>

   <xslt:param name="var7" as="xs:int">
      <item>131072</item>
   </xslt:param>

   <xslt:param name="var8" as="xs:nonPositiveInteger">
      <item>-7632460</item>
   </xslt:param>

   <xslt:param name="var9" as="xs:positiveInteger">
      <item>18446744</item>
   </xslt:param>

   <xslt:param name="var10" as="xs:nonNegativeInteger">
      <item>4444</item>
   </xslt:param>

   <xslt:param name="var11" as="xs:negativeInteger">
      <item>-74146</item>
   </xslt:param>

   <xslt:param name="var12" as="xs:long">
      <item>-9223372</item>
   </xslt:param>

   <xslt:param name="var13" as="xs:unsignedLong">
      <item>1844674</item>
   </xslt:param>

   <xslt:param name="var14" as="xs:unsignedInt">
      <item>573231403</item>
   </xslt:param>

   <xslt:param name="var15" as="xs:short">
      <item>-32768</item>
   </xslt:param>

   <xslt:param name="var16" as="xs:unsignedShort">
      <item>65535</item>
   </xslt:param>

   <xslt:param name="var17" as="xs:byte">
      <item>-128</item>
   </xslt:param>

   <xslt:param name="var18" as="xs:unsignedByte">
      <item>110</item>
   </xslt:param>

   <xslt:param name="var19" as="xs:ID">
      <item>_a</item>
   </xslt:param>

   <xslt:param name="var20" as="xs:IDREF">
      <item>_a</item>
   </xslt:param>

   <xslt:template match="/doc">
      <out>
         <var1 val="{$var1}">
            <xslt:value-of select="$var1 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var1 instance of xs:normalizedString"/>
         </var1>
         <var2 val="{$var2}">
            <xslt:value-of select="$var2 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var2 instance of xs:token"/>
         </var2>
         <var3 val="{$var3}">
            <xslt:value-of select="$var3 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var3 instance of xs:language"/>
         </var3>
         <var4 val="{$var4}">
            <xslt:value-of select="$var4 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var4 instance of xs:Name"/>
         </var4>
         <var5 val="{$var5}">
            <xslt:value-of select="$var5 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var5 instance of xs:NCName"/>
         </var5>
         <var6 val="{$var6}">
            <xslt:value-of select="$var6 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var6 instance of xs:NMTOKEN"/>
         </var6>
         <var7 val="{$var7}">
            <xslt:value-of select="$var7 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var7 instance of xs:int"/>
         </var7>
         <var8 val="{$var8}">
            <xslt:value-of select="$var8 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var8 instance of xs:nonPositiveInteger"/>
         </var8>
         <var9 val="{$var9}">
            <xslt:value-of select="$var9 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var9 instance of xs:positiveInteger"/>
         </var9>
         <var10 val="{$var10}">
            <xslt:value-of select="$var10 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var10 instance of xs:nonNegativeInteger"/>
         </var10>
         <var11 val="{$var11}">
            <xslt:value-of select="$var11 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var11 instance of xs:negativeInteger"/>
         </var11>
         <var12 val="{$var12}">
            <xslt:value-of select="$var12 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var12 instance of xs:long"/>
         </var12>
         <var13 val="{$var13}">
            <xslt:value-of select="$var13 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var13 instance of xs:unsignedLong"/>
         </var13>
         <var14 val="{$var14}">
            <xslt:value-of select="$var14 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var14 instance of xs:unsignedInt"/>
         </var14>
         <var15 val="{$var15}">
            <xslt:value-of select="$var15 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var15 instance of xs:short"/>
         </var15>
         <var16 val="{$var16}">
            <xslt:value-of select="$var16 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var16 instance of xs:unsignedShort"/>
         </var16>
         <var17 val="{$var17}">
            <xslt:value-of select="$var17 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var17 instance of xs:byte"/>
         </var17>
         <var18 val="{$var18}">
            <xslt:value-of select="$var18 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var18 instance of xs:unsignedByte"/>
         </var18>
         <var19 val="{$var19}">
            <xslt:value-of select="$var19 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var19 instance of xs:ID"/>
         </var19>
         <var20 val="{$var20}">
            <xslt:value-of select="$var20 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var20 instance of xs:IDREF"/>
         </var20>
      </out>
   </xslt:template>
</xslt:transform>
