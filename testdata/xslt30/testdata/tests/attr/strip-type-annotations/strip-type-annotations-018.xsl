<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                input-type-annotations="strip"
                version="2.0">
<!-- Purpose: Show that when @input-type-annotations="strip" the type of all attribute nodes is changed to xs:untypedAtomic. 
  				Only built-in schema types are used.-->

   <xslt:template match="/doc-striptype">
      
      <out>
         
         <E1>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-anyURI) instance of xs:anyURI"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-anyURI instance of attribute(*, xs:anyURI)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-anyURI) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-anyURI instance of attribute(*, xs:untypedAtomic)"/>
         </E1>
         
         <E2>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-base64Binary) instance of xs:base64Binary"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-base64Binary instance of attribute(*, xs:base64Binary)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-base64Binary) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-base64Binary instance of attribute(*, xs:untypedAtomic)"/>
         </E2>
         
         <E3>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-boolean) instance of xs:boolean"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-boolean instance of attribute(*, xs:boolean)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-boolean) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-boolean instance of attribute(*, xs:untypedAtomic)"/>
         </E3>
         
         <E4>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-date) instance of xs:date"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-date instance of attribute(*, xs:date)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-date) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-date instance of attribute(*, xs:untypedAtomic)"/>
         </E4>
         
         <E5>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-dateTime) instance of xs:dateTime"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-dateTime instance of attribute(*, xs:dateTime)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-dateTime) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-dateTime instance of attribute(*, xs:untypedAtomic)"/>
         </E5>
         
         <E6>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-decimal) instance of xs:decimal"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-decimal instance of attribute(*, xs:decimal)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-decimal) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-decimal instance of attribute(*, xs:untypedAtomic)"/>
         </E6>
         
         <E7>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-double) instance of xs:double"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-double instance of attribute(*, xs:double)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-double) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-double instance of attribute(*, xs:untypedAtomic)"/>
         </E7>
         
         <E8>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-duration) instance of xs:duration"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-duration instance of attribute(*, xs:duration)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-duration) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-duration instance of attribute(*, xs:untypedAtomic)"/>
         </E8>
         
         <E9>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-float) instance of xs:float"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-float instance of attribute(*, xs:float)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-float) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-float instance of attribute(*, xs:untypedAtomic)"/>
         </E9>
         
         <E10>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-gDay) instance of xs:gDay"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-gDay instance of attribute(*, xs:gDay)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-gDay) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-gDay instance of attribute(*, xs:untypedAtomic)"/>
         </E10>
         
         <E11>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-gMonth) instance of xs:gMonth"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-gMonth instance of attribute(*, xs:gMonth)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-gMonth) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-gMonth instance of attribute(*, xs:untypedAtomic)"/>
         </E11>
         
         <E12>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-gMonthDay) instance of xs:gMonthDay"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-gMonthDay instance of attribute(*, xs:gMonthDay)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-gMonthDay) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-gMonthDay instance of attribute(*, xs:untypedAtomic)"/>
         </E12>
         
         <E13>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-gYear) instance of xs:gYear"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-gYear instance of attribute(*, xs:gYear)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-gYear) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-gYear instance of attribute(*, xs:untypedAtomic)"/>
         </E13>
         
         <E14>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-gYearMonth) instance of xs:gYearMonth"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-gYearMonth instance of attribute(*, xs:gYearMonth)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-gYearMonth) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-gYearMonth instance of attribute(*, xs:untypedAtomic)"/>
         </E14>
         
         <E15>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-hexBinary) instance of xs:hexBinary"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-hexBinary instance of attribute(*, xs:hexBinary)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-hexBinary) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-hexBinary instance of attribute(*, xs:untypedAtomic)"/>
         </E15>
         
         <E16>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-integer) instance of xs:integer"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-integer instance of attribute(*, xs:integer)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-integer) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-integer instance of attribute(*, xs:untypedAtomic)"/>
         </E16>
         
         <E17>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-QName) instance of xs:QName"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-QName instance of attribute(*, xs:QName)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-QName) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-QName instance of attribute(*, xs:untypedAtomic)"/>
         </E17>
         
         <E18>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-string) instance of xs:string"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-string instance of attribute(*, xs:string)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-string) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-string instance of attribute(*, xs:untypedAtomic)"/>
         </E18>
         
         <E19>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-time) instance of xs:time"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-time instance of attribute(*, xs:time)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-time) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-time instance of attribute(*, xs:untypedAtomic)"/>
         </E19>
         
         <E20>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-normalizedString) instance of xs:normalizedString"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-normalizedString instance of attribute(*, xs:normalizedString)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-normalizedString) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-normalizedString instance of attribute(*, xs:untypedAtomic)"/>
         </E20>
         
         <E21>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-token) instance of xs:token"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-token instance of attribute(*, xs:token)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-token) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-token instance of attribute(*, xs:untypedAtomic)"/>
         </E21>
         
         <E22>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-language) instance of xs:language"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-language instance of attribute(*, xs:language)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-language) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-language instance of attribute(*, xs:untypedAtomic)"/>
         </E22>
         
         <E23>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-Name) instance of xs:Name"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-Name instance of attribute(*, xs:Name)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-Name) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-Name instance of attribute(*, xs:untypedAtomic)"/>
         </E23>
         
         <E24>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-NCName) instance of xs:NCName"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-NCName instance of attribute(*, xs:NCName)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-NCName) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-NCName instance of attribute(*, xs:untypedAtomic)"/>
         </E24>
         
         <E25>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-NMTOKEN) instance of xs:NMTOKEN"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-NMTOKEN instance of attribute(*, xs:NMTOKEN)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-NMTOKEN) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-NMTOKEN instance of attribute(*, xs:untypedAtomic)"/>
         </E25>
         
         <E26>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-int) instance of xs:int"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-int instance of attribute(*, xs:int)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-int) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-int instance of attribute(*, xs:untypedAtomic)"/>
         </E26>
         
         <E27>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-nonPositiveInteger) instance of xs:nonPositiveInteger"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-nonPositiveInteger instance of attribute(*, xs:nonPositiveInteger)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-nonPositiveInteger) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-nonPositiveInteger instance of attribute(*, xs:untypedAtomic)"/>
         </E27>
         
         <E28>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-positiveInteger) instance of xs:positiveInteger"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-positiveInteger instance of attribute(*, xs:positiveInteger)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-positiveInteger) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-positiveInteger instance of attribute(*, xs:untypedAtomic)"/>
         </E28>
         
         <E29>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-nonNegativeInteger) instance of xs:nonNegativeInteger"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-nonNegativeInteger instance of attribute(*, xs:nonNegativeInteger)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-nonNegativeInteger) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-nonNegativeInteger instance of attribute(*, xs:untypedAtomic)"/>
         </E29>
         
         <E30>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-negativeInteger) instance of xs:negativeInteger"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-negativeInteger instance of attribute(*, xs:negativeInteger)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-negativeInteger) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-negativeInteger instance of attribute(*, xs:untypedAtomic)"/>
         </E30>
         
         <E31>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-long) instance of xs:long"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-long instance of attribute(*, xs:long)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-long) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-long instance of attribute(*, xs:untypedAtomic)"/>
         </E31>
         
         <E32>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-unsignedLong) instance of xs:unsignedLong"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-unsignedLong instance of attribute(*, xs:unsignedLong)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-unsignedLong) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-unsignedLong instance of attribute(*, xs:untypedAtomic)"/>
         </E32>
         
         <E33>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-unsignedInt) instance of xs:unsignedInt"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-unsignedInt instance of attribute(*, xs:unsignedInt)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-unsignedInt) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-unsignedInt instance of attribute(*, xs:untypedAtomic)"/>
         </E33>
         
         <E34>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-short) instance of xs:short"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-short instance of attribute(*, xs:short)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-short) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-short instance of attribute(*, xs:untypedAtomic)"/>
         </E34>
         
         <E35>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-unsignedShort) instance of xs:unsignedShort"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-unsignedShort instance of attribute(*, xs:unsignedShort)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-unsignedShort) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-unsignedShort instance of attribute(*, xs:untypedAtomic)"/>
         </E35>
         
         <E36>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-byte) instance of xs:byte"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-byte instance of attribute(*, xs:byte)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-byte) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-byte instance of attribute(*, xs:untypedAtomic)"/>
         </E36>
         
         <E37>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-unsignedByte) instance of xs:unsignedByte"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-unsignedByte instance of attribute(*, xs:unsignedByte)"/>
            <xslt:value-of select="data(elem-ALL-attr-Types/@attr-unsignedByte) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ALL-attr-Types/@attr-unsignedByte instance of attribute(*, xs:untypedAtomic)"/>
         </E37>
         
         <E38>
            <xslt:value-of select="data(elem-ID-attr/@attr1) instance of xs:ID"/>
            <xslt:value-of select="elem-ID-attr/@attr1 instance of attribute(*, xs:ID)"/>
            <xslt:value-of select="data(elem-ID-attr/@attr1) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ID-attr/@attr1 instance of attribute(*, xs:untypedAtomic)"/>
         </E38>
         
         <E39>
            <xslt:value-of select="data(elem-ID-attr/@attr2) instance of xs:integer"/>
            <xslt:value-of select="elem-ID-attr/@attr2 instance of attribute(*, xs:integer)"/>
            <xslt:value-of select="data(elem-ID-attr/@attr2) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-ID-attr/@attr2 instance of attribute(*, xs:untypedAtomic)"/>
         </E39>
         
         <E40>
            <xslt:value-of select="data(elem-IDREF-attr/@attr1) instance of xs:IDREF"/>
            <xslt:value-of select="elem-IDREF-attr/@attr1 instance of attribute(*, xs:IDREF)"/>
            <xslt:value-of select="data(elem-IDREF-attr/@attr1) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-IDREF-attr/@attr1 instance of attribute(*, xs:untypedAtomic)"/>
         </E40>
         
         <E41>
            <xslt:value-of select="data(elem-IDREF-attr/@attr2) instance of xs:byte"/>
            <xslt:value-of select="elem-IDREF-attr/@attr2 instance of attribute(*, xs:byte)"/>
            <xslt:value-of select="data(elem-IDREF-attr/@attr2) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-IDREF-attr/@attr2 instance of attribute(*, xs:untypedAtomic)"/>
         </E41>
      </out>
   </xslt:template>
</xslt:transform>
