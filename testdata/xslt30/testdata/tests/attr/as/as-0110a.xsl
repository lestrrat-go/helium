<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.mytest.net"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test with global xsl:variable where value in @select is explicitly 
  				constructed and of the same type as the built-in atomic type in @as. 
  				Types tested are: all built-in primitive types, xs:integer, 
				xs:dayTimeDuration, xs:yearMonthDuration  -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="var1"
                  select="xs:duration('P1Y2M3DT10H30M23S')"
                  as="xs:duration"/>

   <xslt:variable name="var2"
                  select="xs:duration('-P12M23DT0M59.123S')"
                  as="xs:duration"/>

   <xslt:variable name="var3"
                  select="xs:dateTime('2000-01-16T00:00:00Z')"
                  as="xs:dateTime"/>

   <xslt:variable name="var4"
                  select="xs:dateTime('-1999-05-31T13:20:00+14:00')"
                  as="xs:dateTime"/>

   <xslt:variable name="var5" select="xs:time('13:20:00-05:00')" as="xs:time"/>

   <xslt:variable name="var6" select="xs:time('00:00:00')" as="xs:time"/>

   <xslt:variable name="var7" select="xs:date('-0012-12-03-05:00')" as="xs:date"/>

   <xslt:variable name="var8" select="xs:date('1999-05-31')" as="xs:date"/>

   <xslt:variable name="var9"
                  select="xs:gYearMonth('21999-05+14:00')"
                  as="xs:gYearMonth"/>

   <xslt:variable name="var10" select="xs:gYearMonth('1999-05')" as="xs:gYearMonth"/>

   <xslt:variable name="var11" select="xs:gYear('-0012-05:00')" as="xs:gYear"/>

   <xslt:variable name="var12" select="xs:gYear('2005')" as="xs:gYear"/>

   <xslt:variable name="var13" select="xs:gMonthDay('--05-31')" as="xs:gMonthDay"/>

   <xslt:variable name="var14"
                  select="xs:gMonthDay('--05-31+14:00')"
                  as="xs:gMonthDay"/>

   <xslt:variable name="var15" select="xs:gDay('---31+14:00')" as="xs:gDay"/>

   <xslt:variable name="var16" select="xs:gDay('---31Z')" as="xs:gDay"/>

   <xslt:variable name="var17" select="xs:gMonth('--05+14:00')" as="xs:gMonth"/>

   <xslt:variable name="var18" select="xs:gMonth('--05')" as="xs:gMonth"/>

   <xslt:variable name="var19" select="xs:boolean('true')" as="xs:boolean"/>

   <xslt:variable name="var20" select="xs:base64Binary('PM24')" as="xs:base64Binary"/>

   <xslt:variable name="var21" select="xs:hexBinary('07FF')" as="xs:hexBinary"/>

   <xslt:variable name="var22" select="xs:float(1.1234E99)" as="xs:float"/>

   <xslt:variable name="var23" select="xs:float(-1.75e-3)" as="xs:float"/>

   <xslt:variable name="var24" select="xs:decimal(-3.420100)" as="xs:decimal"/>

   <xslt:variable name="var25" select="xs:decimal(7.8899)" as="xs:decimal"/>

   <xslt:variable name="var26" select="xs:decimal(0.0)" as="xs:decimal"/>

   <xslt:variable name="var27" select="xs:double(5.4321E-1001)" as="xs:double"/>

   <xslt:variable name="var28" select="xs:double(-0.0E0)" as="xs:double"/>

   <xslt:variable name="var29" select="xs:double('INF')" as="xs:double"/>

   <xslt:variable name="var30" select="xs:integer(43)" as="xs:integer"/>

   <xslt:variable name="var31" select="xs:integer(-300)" as="xs:integer"/>

   <xslt:variable name="var32"
                  select="xs:anyURI('http://www.mytypetest.org')"
                  as="xs:anyURI"/>

   <xslt:variable name="var33" select="xs:QName('my:local')" as="xs:QName"/>

   <xslt:variable name="var34"
                  select="xs:dayTimeDuration('PT99.999S')"
                  as="xs:dayTimeDuration"/>

   <xslt:variable name="var35"
                  select="xs:dayTimeDuration('-PT100M')"
                  as="xs:dayTimeDuration"/>

   <xslt:variable name="var36"
                  select="xs:yearMonthDuration('P1Y2M')"
                  as="xs:yearMonthDuration"/>

   <xslt:variable name="var37"
                  select="xs:yearMonthDuration('-P21M')"
                  as="xs:yearMonthDuration"/>

   <xslt:variable name="var38" select="xs:string('hello')" as="xs:string"/>

   <xslt:template match="/doc">
      <out>
         
         <var1>
            <xslt:value-of select="$var1 instance of xs:duration"/>
         </var1>
         
         <var2>
            <xslt:value-of select="$var2 instance of xs:duration"/>
         </var2>
         
         <var3>
            <xslt:value-of select="$var3 instance of xs:dateTime"/>
         </var3>
         
         <var4>
            <xslt:value-of select="$var4 instance of xs:dateTime"/>
         </var4>
         
         <var5>
            <xslt:value-of select="$var5 instance of xs:time"/>
         </var5>
         
         <var6>
            <xslt:value-of select="$var6 instance of xs:time"/>
         </var6>
         
         <var7>
            <xslt:value-of select="$var7 instance of xs:date"/>
         </var7>
         
         <var8>
            <xslt:value-of select="$var8 instance of xs:date"/>
         </var8>
         
         <var9>
            <xslt:value-of select="$var9 instance of xs:gYearMonth"/>
         </var9>
         
         <var10>
            <xslt:value-of select="$var10 instance of xs:gYearMonth"/>
         </var10>
         
         <var11>
            <xslt:value-of select="$var11 instance of xs:gYear"/>
         </var11>
         
         <var12>
            <xslt:value-of select="$var12 instance of xs:gYear"/>
         </var12>
         
         <var13>
            <xslt:value-of select="$var13 instance of xs:gMonthDay"/>
         </var13>
         
         <var14>
            <xslt:value-of select="$var14 instance of xs:gMonthDay"/>
         </var14>
         
         <var15>
            <xslt:value-of select="$var15 instance of xs:gDay"/>
         </var15>
         
         <var16>
            <xslt:value-of select="$var16 instance of xs:gDay"/>
         </var16>
         
         <var17>
            <xslt:value-of select="$var17 instance of xs:gMonth"/>
         </var17>
         
         <var18>
            <xslt:value-of select="$var18 instance of xs:gMonth"/>
         </var18>
         
         <var19>
            <xslt:value-of select="$var19 instance of xs:boolean"/>
         </var19>
         
         <var20>
            <xslt:value-of select="$var20 instance of xs:base64Binary"/>
         </var20>
         
         <var21>
            <xslt:value-of select="$var21 instance of xs:hexBinary"/>
         </var21>
         
         <var22>
            <xslt:value-of select="$var22 instance of xs:float"/>
         </var22>
         
         <var23>
            <xslt:value-of select="$var23 instance of xs:float"/>
         </var23>
         
         <var24>
            <xslt:value-of select="$var24 instance of xs:decimal"/>
         </var24>
         
         <var25>
            <xslt:value-of select="$var25 instance of xs:decimal"/>
         </var25>
         
         <var26>
            <xslt:value-of select="$var26 instance of xs:decimal"/>
         </var26>
         
         <var27>
            <xslt:value-of select="$var27 instance of xs:double"/>
         </var27>
         
         <var28>
            <xslt:value-of select="$var28 instance of xs:double"/>
         </var28>
         
         <var29>
            <xslt:value-of select="$var29 instance of xs:double"/>
         </var29>
         
         <var30>
            <xslt:value-of select="$var30 instance of xs:integer"/>
         </var30>
         
         <var31>
            <xslt:value-of select="$var31 instance of xs:integer"/>
         </var31>
         
         <var32>
            <xslt:value-of select="$var32 instance of xs:anyURI"/>
         </var32>
         
         <var33>
            <xslt:value-of select="$var33 instance of xs:QName"/>
         </var33>
         
         <var34>
            <xslt:value-of select="$var34 instance of xs:dayTimeDuration"/>
         </var34>
         
         <var35>
            <xslt:value-of select="$var35 instance of xs:dayTimeDuration"/>
         </var35>
         
         <var36>
            <xslt:value-of select="$var36 instance of xs:yearMonthDuration"/>
         </var36>
         
         <var37>
            <xslt:value-of select="$var37 instance of xs:yearMonthDuration"/>
         </var37>
         
         <var38>
            <xslt:value-of select="$var38 instance of xs:string"/>
         </var38>
      </out>
   </xslt:template>
</xslt:transform>
