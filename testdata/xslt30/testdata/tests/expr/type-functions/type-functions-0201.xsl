<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.schemafunc.example.com/ns/various"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test fn:abs with typed numeric arguments coming from nodes in input source. Types tested:
   				- xs:float, xs:double, xs:decimal, xs:integer, xs:nonPositiveInteger, xs:long, xs:negativeInteger
   				xs:positiveInteger, xs:short, xs:int; xs:unsignedInt;
   				- user-defined types derived by restriction from the types listed above -->

   <xslt:import-schema namespace="http://www.schemafunc.example.com/ns/various"
      schema-location="variousTypesSchemaFunc.xsd"/>

   <xslt:template match="/my:userNode">
      <out>
         <e1>
            <xslt:value-of select="data(elem-decimal) instance of xs:decimal"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-decimal)"/>
         </e1>
         <e2>
            <xslt:value-of select="data(elem-double) instance of xs:double"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-double)"/>
         </e2>
         <e3>
            <xslt:value-of select="data(elem-float) instance of xs:float"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-float)"/>
         </e3>
         <e4>
            <xslt:value-of select="data(elem-integer) instance of xs:integer"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-integer)"/>
         </e4>
         <e5>
            <xslt:value-of select="data(elem-int) instance of xs:int"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-int)"/>
         </e5>
         <e6>
            <xslt:value-of select="data(elem-nonPositiveInteger) instance of xs:nonPositiveInteger"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-nonPositiveInteger)"/>
         </e6>
         <e7>
            <xslt:value-of select="data(elem-positiveInteger) instance of xs:positiveInteger"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-positiveInteger)"/>
         </e7>
         <e8>
            <xslt:value-of select="data(elem-negativeInteger) instance of xs:negativeInteger"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-negativeInteger)"/>
         </e8>
         <e9>
            <xslt:value-of select="data(elem-long) instance of xs:long"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-long)"/>
         </e9>
         <e10>
            <xslt:value-of select="data(elem-unsignedInt) instance of xs:unsignedInt"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-unsignedInt)"/>
         </e10>
         <e11>
            <xslt:value-of select="data(elem-short) instance of xs:short"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-short)"/>
         </e11>
         <e12>
            <xslt:value-of select="data(elem-decimal-derived) instance of xs:decimal"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-decimal-derived) instance of my:elem-decimal-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-decimal-derived)"/>
         </e12>
         <e13>
            <xslt:value-of select="data(elem-double-derived) instance of xs:double"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-double-derived) instance of my:elem-double-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-double-derived)"/>
         </e13>
         <e14>
            <xslt:value-of select="data(elem-float-derived) instance of xs:float"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="data(elem-float-derived) instance of my:elem-float-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-float-derived)"/>
         </e14>
         <e15>
            <xslt:value-of select="data(elem-integer-derived) instance of xs:integer"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-integer-derived) instance of my:elem-integer-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-integer-derived)"/>
         </e15>
         <e16>
            <xslt:value-of select="data(elem-int-derived) instance of xs:int"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="data(elem-int-derived) instance of my:elem-int-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-int-derived)"/>
         </e16>
         <e17>
            <xslt:value-of
               select="data(elem-nonPositiveInteger-derived) instance of xs:nonPositiveInteger"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-nonPositiveInteger-derived) instance of my:elem-nonPositiveInteger-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-nonPositiveInteger-derived)"/>
         </e17>
         <e18>
            <xslt:value-of
               select="data(elem-positiveInteger-derived) instance of xs:positiveInteger"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-positiveInteger-derived) instance of my:elem-positiveInteger-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-positiveInteger-derived)"/>
         </e18>
         <e19>
            <xslt:value-of
               select="data(elem-negativeInteger-derived) instance of xs:negativeInteger"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-negativeInteger-derived) instance of my:elem-negativeInteger-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-negativeInteger-derived)"/>
         </e19>
         <e20>
            <xslt:value-of select="data(elem-long-derived) instance of xs:long"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="data(elem-long-derived) instance of my:elem-long-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-long-derived)"/>
         </e20>
         <e21>
            <xslt:value-of select="data(elem-unsignedInt-derived) instance of xs:unsignedInt"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-unsignedInt-derived) instance of my:elem-unsignedInt-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-unsignedInt-derived)"/>
         </e21>
         <e22>
            <xslt:value-of select="data(elem-short-derived) instance of xs:short"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="data(elem-short-derived) instance of my:elem-short-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="abs(elem-short-derived)"/>
         </e22>
      </out>
   </xslt:template>
</xslt:transform>
