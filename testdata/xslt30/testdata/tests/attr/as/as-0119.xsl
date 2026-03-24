<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.mytest.net"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with an empty sequence obtained at run time in @select of global xsl:variable and @as 
  				is a built-in atomic type with occurence indicator (?). Verify the variable is of the 
  				type specified in @as.Values for @as tested: all built-in primitive types (except xs:QName), 
  				xs:integer, xs:dayTimeDuration, xs:yearMonthDuration  -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="var1" select="/doc/item1" as="xs:duration?"/>

   <xslt:variable name="var3" select="/doc/item3" as="xs:dateTime?"/>

   <xslt:variable name="var5" select="/doc/item5" as="xs:time?"/>

   <xslt:variable name="var7" select="/doc/item7" as="xs:date?"/>

   <xslt:variable name="var9" select="/doc/item9" as="xs:gYearMonth?"/>

   <xslt:variable name="var11" select="/doc/item11" as="xs:gYear?"/>

   <xslt:variable name="var13" select="/doc/item13" as="xs:gMonthDay?"/>

   <xslt:variable name="var15" select="/doc/item15" as="xs:gDay?"/>

   <xslt:variable name="var17" select="/doc/item17" as="xs:gMonth?"/>

   <xslt:variable name="var19" select="/doc/item19" as="xs:boolean?"/>

   <xslt:variable name="var20" select="/doc/item20" as="xs:base64Binary?"/>

   <xslt:variable name="var21" select="/doc/item21" as="xs:hexBinary?"/>

   <xslt:variable name="var22" select="/doc/item22" as="xs:float?"/>

   <xslt:variable name="var24" select="/doc/item24" as="xs:decimal?"/>

   <xslt:variable name="var27" select="/doc/item27" as="xs:double?"/>

   <xslt:variable name="var30" select="/doc/item30" as="xs:integer?"/>

   <xslt:variable name="var32" select="/doc/item32" as="xs:anyURI?"/>

   <xslt:variable name="var34" select="/doc/item34" as="xs:dayTimeDuration?"/>

   <xslt:variable name="var36" select="/doc/item36" as="xs:yearMonthDuration?"/>

   <xslt:variable name="var38" select="/doc/item38" as="xs:string?"/>

   <xslt:template match="/doc">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of xs:duration?"/>
            <xslt:value-of select="$var1 instance of xs:duration"/>
         </var1>
         <var3>
            <xslt:value-of select="$var3 instance of xs:dateTime?"/>
            <xslt:value-of select="$var3 instance of xs:dateTime"/>
         </var3>
         <var5>
            <xslt:value-of select="$var5 instance of xs:time?"/>
            <xslt:value-of select="$var5 instance of xs:time"/>
         </var5>
         <var7>
            <xslt:value-of select="$var7 instance of xs:date?"/>
            <xslt:value-of select="$var7 instance of xs:date"/>
         </var7>
         <var9>
            <xslt:value-of select="$var9 instance of xs:gYearMonth?"/>
            <xslt:value-of select="$var9 instance of xs:gYearMonth"/>
         </var9>
         <var11>
            <xslt:value-of select="$var11 instance of xs:gYear?"/>
            <xslt:value-of select="$var11 instance of xs:gYear"/>
         </var11>
         <var13>
            <xslt:value-of select="$var13 instance of xs:gMonthDay?"/>
            <xslt:value-of select="$var13 instance of xs:gMonthDay"/>
         </var13>
         <var15>
            <xslt:value-of select="$var15 instance of xs:gDay?"/>
            <xslt:value-of select="$var15 instance of xs:gDay"/>
         </var15>
         <var17>
            <xslt:value-of select="$var17 instance of xs:gMonth?"/>
            <xslt:value-of select="$var17 instance of xs:gMonth"/>
         </var17>
         <var19>
            <xslt:value-of select="$var19 instance of xs:boolean?"/>
            <xslt:value-of select="$var19 instance of xs:boolean"/>
         </var19>
         <var20>
            <xslt:value-of select="$var20 instance of xs:base64Binary?"/>
            <xslt:value-of select="$var20 instance of xs:base64Binary"/>
         </var20>
         <var21>
            <xslt:value-of select="$var21 instance of xs:hexBinary?"/>
            <xslt:value-of select="$var21 instance of xs:hexBinary"/>
         </var21>
         <var22>
            <xslt:value-of select="$var22 instance of xs:float?"/>
            <xslt:value-of select="$var22 instance of xs:float"/>
         </var22>
         <var24>
            <xslt:value-of select="$var24 instance of xs:decimal?"/>
            <xslt:value-of select="$var24 instance of xs:decimal"/>
         </var24>
         <var27>
            <xslt:value-of select="$var27 instance of xs:double?"/>
            <xslt:value-of select="$var27 instance of xs:double"/>
         </var27>
         <var30>
            <xslt:value-of select="$var30 instance of xs:integer?"/>
            <xslt:value-of select="$var30 instance of xs:integer"/>
         </var30>
         <var32>
            <xslt:value-of select="$var32 instance of xs:anyURI?"/>
            <xslt:value-of select="$var32 instance of xs:anyURI"/>
         </var32>
         <var34>
            <xslt:value-of select="$var34 instance of xs:dayTimeDuration?"/>
            <xslt:value-of select="$var34 instance of xs:dayTimeDuration"/>
         </var34>
         <var36>
            <xslt:value-of select="$var36 instance of xs:yearMonthDuration?"/>
            <xslt:value-of select="$var36 instance of xs:yearMonthDuration"/>
         </var36>
         <var38>
            <xslt:value-of select="$var38 instance of xs:string?"/>
            <xslt:value-of select="$var38 instance of xs:string"/>
         </var38>
      </out>
   </xslt:template>
</xslt:transform>
