<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://testnamesapce.uri"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with global xsl:variable without @select, 
  				where sequence constructor contains an explicitly 
  				constructed typed value of the same type as the built-in 
  				derived atomic type in @as. Verify the variable is of the 
  				type specified in @as. -->

   <xslt:variable name="var1" as="xs:normalizedString">
      <xslt:value-of select="xs:normalizedString('like,')"/>
   </xslt:variable>

   <xslt:variable name="var2" as="xs:token">
      <xslt:value-of select="xs:token('two organizations')"/>
   </xslt:variable>

   <xslt:variable name="var3" as="xs:language">
      <xslt:value-of select="xs:language('PT')"/>
   </xslt:variable>

   <xslt:variable name="var4" as="xs:Name">
      <xslt:value-of select="xs:Name('a:_underscore_')"/>
   </xslt:variable>

   <xslt:variable name="var5" as="xs:NCName">
      <xslt:value-of select="xs:NCName('vwxyz12')"/>
   </xslt:variable>

   <xslt:variable name="var6" as="xs:NMTOKEN">
      <xslt:value-of select="xs:NMTOKEN('business-enforcement::link-chain.common')"/>
   </xslt:variable>

   <xslt:variable name="var7" as="xs:int">
      <xslt:value-of select="xs:int(131072)"/>
   </xslt:variable>

   <xslt:variable name="var8" as="xs:nonPositiveInteger">
      <xslt:value-of select="xs:nonPositiveInteger(-7632460)"/>
   </xslt:variable>

   <xslt:variable name="var9" as="xs:positiveInteger">
      <xslt:value-of select="xs:positiveInteger(18446744)"/>
   </xslt:variable>

   <xslt:variable name="var10" as="xs:nonNegativeInteger">
      <xslt:value-of select="xs:nonNegativeInteger(4444)"/>
   </xslt:variable>

   <xslt:variable name="var11" as="xs:negativeInteger">
      <xslt:value-of select="xs:negativeInteger(-74146)"/>
   </xslt:variable>

   <xslt:variable name="var12" as="xs:long">
      <xslt:value-of select="xs:long(-9223372)"/>
   </xslt:variable>

   <xslt:variable name="var13" as="xs:unsignedLong">
      <xslt:value-of select="xs:unsignedLong(1844674)"/>
   </xslt:variable>

   <xslt:variable name="var14" as="xs:unsignedInt">
      <xslt:value-of select="xs:unsignedInt(573231403)"/>
   </xslt:variable>

   <xslt:variable name="var15" as="xs:short">
      <xslt:value-of select="xs:short(-32768)"/>
   </xslt:variable>

   <xslt:variable name="var16" as="xs:unsignedShort">
      <xslt:value-of select="xs:unsignedShort(65535)"/>
   </xslt:variable>

   <xslt:variable name="var17" as="xs:byte">
      <xslt:value-of select="xs:byte(-128)"/>
   </xslt:variable>

   <xslt:variable name="var18" as="xs:unsignedByte">
      <xslt:value-of select="xs:unsignedByte(110)"/>
   </xslt:variable>

   <xslt:variable name="var19" as="xs:ID">
      <xslt:value-of select="xs:ID('_a')"/>
   </xslt:variable>

   <xslt:variable name="var20" as="xs:IDREF">
      <xslt:value-of select="xs:IDREF('_a')"/>
   </xslt:variable>

   <xslt:template match="/doc">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var1 instance of xs:normalizedString"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var2 instance of xs:token"/>
         </var2>
         <var3>
            <xslt:value-of select="$var3 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var3 instance of xs:language"/>
         </var3>
         <var4>
            <xslt:value-of select="$var4 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var4 instance of xs:Name"/>
         </var4>
         <var5>
            <xslt:value-of select="$var5 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var5 instance of xs:NCName"/>
         </var5>
         <var6>
            <xslt:value-of select="$var6 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var6 instance of xs:NMTOKEN"/>
         </var6>
         <var7>
            <xslt:value-of select="$var7 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var7 instance of xs:int"/>
         </var7>
         <var8>
            <xslt:value-of select="$var8 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var8 instance of xs:nonPositiveInteger"/>
         </var8>
         <var9>
            <xslt:value-of select="$var9 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var9 instance of xs:positiveInteger"/>
         </var9>
         <var10>
            <xslt:value-of select="$var10 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var10 instance of xs:nonNegativeInteger"/>
         </var10>
         <var11>
            <xslt:value-of select="$var11 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var11 instance of xs:negativeInteger"/>
         </var11>
         <var12>
            <xslt:value-of select="$var12 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var12 instance of xs:long"/>
         </var12>
         <var13>
            <xslt:value-of select="$var13 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var13 instance of xs:unsignedLong"/>
         </var13>
         <var14>
            <xslt:value-of select="$var14 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var14 instance of xs:unsignedInt"/>
         </var14>
         <var15>
            <xslt:value-of select="$var15 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var15 instance of xs:short"/>
         </var15>
         <var16>
            <xslt:value-of select="$var16 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var16 instance of xs:unsignedShort"/>
         </var16>
         <var17>
            <xslt:value-of select="$var17 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var17 instance of xs:byte"/>
         </var17>
         <var18>
            <xslt:value-of select="$var18 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var18 instance of xs:unsignedByte"/>
         </var18>
         <var19>
            <xslt:value-of select="$var19 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var19 instance of xs:ID"/>
         </var19>
         <var20>
            <xslt:value-of select="$var20 instance of xs:untypedAtomic"/>
            <xslt:value-of select="$var20 instance of xs:IDREF"/>
         </var20>
      </out>
   </xslt:template>
</xslt:transform>
