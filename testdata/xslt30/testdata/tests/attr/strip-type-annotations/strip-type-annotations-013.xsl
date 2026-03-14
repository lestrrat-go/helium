<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs"
   input-type-annotations="strip" version="2.0">
   <!--     Show that when @input-type-annotations="strip" the type of all element nodes is 
  				changed to xs:untyped. Only built-in schema types are used.-->

   <xslt:template match="/doc-striptype">

      <out>

         <E1>
            <xslt:value-of select="data(elem-anyURI) instance of xs:anyURI"/>
            <xslt:value-of select="elem-anyURI instance of element(*, xs:anyURI)"/>
            <xslt:value-of select="data(elem-anyURI) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-anyURI instance of element(*, xs:untyped)"/>
         </E1>

         <E2>
            <xslt:value-of select="data(elem-base64Binary) instance of xs:base64Binary"/>
            <xslt:value-of select="elem-base64Binary instance of element(*, xs:base64Binary)"/>
            <xslt:value-of select="data(elem-base64Binary) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-base64Binary instance of element(*, xs:untyped)"/>
         </E2>

         <E3>
            <xslt:value-of select="data(elem-boolean) instance of xs:boolean"/>
            <xslt:value-of select="elem-boolean instance of element(*, xs:boolean)"/>
            <xslt:value-of select="data(elem-boolean) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-boolean instance of element(*, xs:untyped)"/>
         </E3>

         <E4>
            <xslt:value-of select="data(elem-date) instance of xs:date"/>
            <xslt:value-of select="elem-date instance of element(*, xs:base64Binary)"/>
            <xslt:value-of select="data(elem-date) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-date instance of element(*, xs:untyped)"/>
         </E4>

         <E5>
            <xslt:value-of select="data(elem-dateTime) instance of xs:dateTime"/>
            <xslt:value-of select="elem-dateTime instance of element(*, xs:dateTime)"/>
            <xslt:value-of select="data(elem-dateTime) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-dateTime instance of element(*, xs:untyped)"/>
         </E5>

         <E6>
            <xslt:value-of select="data(elem-decimal) instance of xs:decimal"/>
            <xslt:value-of select="elem-decimal instance of element(*, xs:decimal)"/>
            <xslt:value-of select="data(elem-decimal) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-decimal instance of element(*, xs:untyped)"/>
         </E6>

         <E7>
            <xslt:value-of select="data(elem-double) instance of xs:double"/>
            <xslt:value-of select="elem-double instance of element(*, xs:double)"/>
            <xslt:value-of select="data(elem-double) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-double instance of element(*, xs:untyped)"/>
         </E7>

         <E8>
            <xslt:value-of select="data(elem-duration) instance of xs:duration"/>
            <xslt:value-of select="elem-duration instance of element(*, xs:duration)"/>
            <xslt:value-of select="data(elem-duration) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-duration instance of element(*, xs:untyped)"/>
         </E8>

         <E9>
            <xslt:value-of select="data(elem-float) instance of xs:float"/>
            <xslt:value-of select="elem-float instance of element(*, xs:float)"/>
            <xslt:value-of select="data(elem-float) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-float instance of element(*, xs:untyped)"/>
         </E9>

         <E10>
            <xslt:value-of select="data(elem-gDay) instance of xs:gDay"/>
            <xslt:value-of select="elem-gDay instance of element(*, xs:gDay)"/>
            <xslt:value-of select="data(elem-gDay) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-gDay instance of element(*, xs:untyped)"/>
         </E10>

         <E11>
            <xslt:value-of select="data(elem-gMonth) instance of xs:gMonth"/>
            <xslt:value-of select="elem-gMonth instance of element(*, xs:gMonth)"/>
            <xslt:value-of select="data(elem-gMonth) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-gMonth instance of element(*, xs:untyped)"/>
         </E11>

         <E12>
            <xslt:value-of select="data(elem-gMonthDay) instance of xs:gMonthDay"/>
            <xslt:value-of select="elem-gMonthDay instance of element(*, xs:gMonthDay)"/>
            <xslt:value-of select="data(elem-gMonthDay) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-gMonthDay instance of element(*, xs:untyped)"/>
         </E12>

         <E13>
            <xslt:value-of select="data(elem-gYear) instance of xs:gYear"/>
            <xslt:value-of select="elem-gYear instance of element(*, xs:gYear)"/>
            <xslt:value-of select="data(elem-gYear) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-gYear instance of element(*, xs:untyped)"/>
         </E13>

         <E14>
            <xslt:value-of select="data(elem-gYearMonth) instance of xs:gYearMonth"/>
            <xslt:value-of select="elem-gYearMonth instance of element(*, xs:gYearMonth)"/>
            <xslt:value-of select="data(elem-gYearMonth) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-gYearMonth instance of element(*, xs:untyped)"/>
         </E14>

         <E15>
            <xslt:value-of select="data(elem-hexBinary) instance of xs:hexBinary"/>
            <xslt:value-of select="elem-hexBinary instance of element(*, xs:hexBinary)"/>
            <xslt:value-of select="data(elem-hexBinary) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-hexBinary instance of element(*, xs:untyped)"/>
         </E15>

         <E16>
            <xslt:value-of select="data(elem-integer) instance of xs:integer"/>
            <xslt:value-of select="elem-integer instance of element(*, xs:integer)"/>
            <xslt:value-of select="data(elem-integer) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-integer instance of element(*, xs:untyped)"/>
         </E16>

         <E17>
            <xslt:value-of select="data(elem-QName) instance of xs:QName"/>
            <xslt:value-of select="elem-QName instance of element(*, xs:QName)"/>
            <xslt:value-of select="data(elem-QName) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-QName instance of element(*, xs:untyped)"/>
         </E17>

         <E18>
            <xslt:value-of select="data(elem-string) instance of xs:string"/>
            <xslt:value-of select="elem-string instance of element(*, xs:string)"/>
            <xslt:value-of select="data(elem-string) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-string instance of element(*, xs:untyped)"/>
         </E18>

         <E19>
            <xslt:value-of select="data(elem-time) instance of xs:time"/>
            <xslt:value-of select="elem-time instance of element(*, xs:time)"/>
            <xslt:value-of select="data(elem-time) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-time instance of element(*, xs:untyped)"/>
         </E19>

         <E20>
            <xslt:value-of select="data(elem-normalizedString) instance of xs:normalizedString"/>
            <xslt:value-of
               select="elem-normalizedString instance of element(*, xs:normalizedString)"/>
            <xslt:value-of select="data(elem-normalizedString) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-normalizedString instance of element(*, xs:untyped)"/>
         </E20>

         <E21>
            <xslt:value-of select="data(elem-token) instance of xs:token"/>
            <xslt:value-of select="elem-token instance of element(*, xs:token)"/>
            <xslt:value-of select="data(elem-token) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-token instance of element(*, xs:untyped)"/>
         </E21>

         <E22>
            <xslt:value-of select="data(elem-language) instance of xs:language"/>
            <xslt:value-of select="elem-language instance of element(*, xs:language)"/>
            <xslt:value-of select="data(elem-language) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-language instance of element(*, xs:untyped)"/>
         </E22>

         <E23>
            <xslt:value-of select="data(elem-Name) instance of xs:Name"/>
            <xslt:value-of select="elem-Name instance of element(*, xs:Name)"/>
            <xslt:value-of select="data(elem-Name) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-Name instance of element(*, xs:untyped)"/>
         </E23>

         <E24>
            <xslt:value-of select="data(elem-NCName) instance of xs:NCName"/>
            <xslt:value-of select="elem-NCName instance of element(*, xs:NCName)"/>
            <xslt:value-of select="data(elem-NCName) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-NCName instance of element(*, xs:untyped)"/>
         </E24>

         <E25>
            <xslt:value-of select="data(elem-NMTOKEN) instance of xs:NMTOKEN"/>
            <xslt:value-of select="elem-NMTOKEN instance of element(*, xs:NMTOKEN)"/>
            <xslt:value-of select="data(elem-NMTOKEN) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-NMTOKEN instance of element(*, xs:untyped)"/>
         </E25>

         <E26>
            <xslt:value-of select="data(elem-int) instance of xs:int"/>
            <xslt:value-of select="elem-int instance of element(*, xs:int)"/>
            <xslt:value-of select="data(elem-int) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-int instance of element(*, xs:untyped)"/>
         </E26>

         <E27>
            <xslt:value-of select="data(elem-nonPositiveInteger) instance of xs:nonPositiveInteger"/>
            <xslt:value-of
               select="elem-nonPositiveInteger instance of element(*, xs:nonPositiveInteger)"/>
            <xslt:value-of select="data(elem-nonPositiveInteger) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-nonPositiveInteger instance of element(*, xs:untyped)"/>
         </E27>

         <E28>
            <xslt:value-of select="data(elem-positiveInteger) instance of xs:positiveInteger"/>
            <xslt:value-of select="elem-positiveInteger instance of element(*, xs:positiveInteger)"/>
            <xslt:value-of select="data(elem-positiveInteger) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-positiveInteger instance of element(*, xs:untyped)"/>
         </E28>

         <E29>
            <xslt:value-of select="data(elem-nonNegativeInteger) instance of xs:nonNegativeInteger"/>
            <xslt:value-of
               select="elem-nonNegativeInteger instance of element(*, xs:nonNegativeInteger)"/>
            <xslt:value-of select="data(elem-nonNegativeInteger) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-nonNegativeInteger instance of element(*, xs:untyped)"/>
         </E29>

         <E30>
            <xslt:value-of select="data(elem-negativeInteger) instance of xs:negativeInteger"/>
            <xslt:value-of select="elem-negativeInteger instance of element(*, xs:negativeInteger)"/>
            <xslt:value-of select="data(elem-negativeInteger) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-negativeInteger instance of element(*, xs:untyped)"/>
         </E30>

         <E31>
            <xslt:value-of select="data(elem-long) instance of xs:long"/>
            <xslt:value-of select="elem-long instance of element(*, xs:long)"/>
            <xslt:value-of select="data(elem-long) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-long instance of element(*, xs:untyped)"/>
         </E31>

         <E32>
            <xslt:value-of select="data(elem-unsignedLong) instance of xs:unsignedLong"/>
            <xslt:value-of select="elem-unsignedLong instance of element(*, xs:unsignedLong)"/>
            <xslt:value-of select="data(elem-unsignedLong) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-unsignedLong instance of element(*, xs:untyped)"/>
         </E32>

         <E33>
            <xslt:value-of select="data(elem-unsignedInt) instance of xs:unsignedInt"/>
            <xslt:value-of select="elem-unsignedInt instance of element(*, xs:unsignedInt)"/>
            <xslt:value-of select="data(elem-unsignedInt) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-unsignedInt instance of element(*, xs:untyped)"/>
         </E33>

         <E34>
            <xslt:value-of select="data(elem-short) instance of xs:short"/>
            <xslt:value-of select="elem-short instance of element(*, xs:short)"/>
            <xslt:value-of select="data(elem-short) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-short instance of element(*, xs:untyped)"/>
         </E34>

         <E35>
            <xslt:value-of select="data(elem-unsignedShort) instance of xs:unsignedShort"/>
            <xslt:value-of select="elem-unsignedShort instance of element(*, xs:unsignedShort)"/>
            <xslt:value-of select="data(elem-unsignedShort) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-unsignedShort instance of element(*, xs:untyped)"/>
         </E35>

         <E36>
            <xslt:value-of select="data(elem-byte) instance of xs:byte"/>
            <xslt:value-of select="elem-byte instance of element(*, xs:byte)"/>
            <xslt:value-of select="data(elem-byte) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-byte instance of element(*, xs:untyped)"/>
         </E36>

         <E37>
            <xslt:value-of select="data(elem-unsignedByte) instance of xs:unsignedByte"/>
            <xslt:value-of select="elem-unsignedByte instance of element(*, xs:unsignedByte)"/>
            <xslt:value-of select="data(elem-unsignedByte) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-unsignedByte instance of element(*, xs:untyped)"/>
         </E37>

      </out>
   </xslt:template>
</xslt:transform>
