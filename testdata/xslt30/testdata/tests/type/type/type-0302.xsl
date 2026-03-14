<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://www.schematype.example.com/ns/various"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                version="2.0">
<!-- Purpose: Test contructor functions for user defined types. The argument is a string/numeric literal -->

   <xslt:import-schema namespace="http://www.schematype.example.com/ns/various"
                       schema-location="variousTypesSchemaType.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:text>
</xslt:text>
         <e1>
            <xslt:value-of select="my:partNumberType('123-AB')"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="my:partNumberType('123-AB') instance of my:partNumberType"/>
         </e1>
         <xslt:text>
</xslt:text>
         <e2>
            <xslt:value-of select="my:derivedURI('http://test1.org')"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="my:derivedURI('http://test1.org') instance of my:derivedURI"/>
         </e2>
         <xslt:text>
</xslt:text>
         <e3>
            <xslt:value-of select="my:specialPartNumber('000-CD')"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="my:specialPartNumber('000-CD') instance of my:specialPartNumber"/>
         </e3>
         <xslt:text>
</xslt:text>
         <e4>
            <xslt:value-of select="my:de1-decimal-enumeration-Inline(-1.01)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="my:de1-decimal-enumeration-Inline(-1.01) instance of my:de1-decimal-enumeration-Inline"/>
         </e4>
         <xslt:text>
</xslt:text>
         <e5>
            <xslt:value-of select="my:de2-decimal-maxExclusive-Inline(123.456788)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="my:de2-decimal-maxExclusive-Inline(123.456788) instance of my:de2-decimal-maxExclusive-Inline"/>
         </e5>
         <xslt:text>
</xslt:text>
         <e6>
            <xslt:value-of select="my:de4-decimal-minInclusive-pattern-Inline(1000.111)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="my:de4-decimal-minInclusive-pattern-Inline(1000.111) instance of my:de4-decimal-minInclusive-pattern-Inline"/>
         </e6>
         <xslt:text>
</xslt:text>
         <e7>
            <xslt:value-of select="my:de5-decimal-fractionDigits-totalDigits-Inline(1234567.1234)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="my:de5-decimal-fractionDigits-totalDigits-Inline(1234567.1234) instance of my:de5-decimal-fractionDigits-totalDigits-Inline"/>
         </e7>
         <xslt:text>
</xslt:text>
         <e8>
            <xslt:value-of select="my:de6-decimal-whiteSpace-Inline(   12    )"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="my:de6-decimal-whiteSpace-Inline(   12    ) instance of my:de6-decimal-whiteSpace-Inline"/>
         </e8>
         <xslt:text>
</xslt:text>
         <e9>
            <xslt:value-of select="my:derived-float-Type(0)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="my:derived-float-Type(0) instance of my:derived-float-Type"/>
         </e9>
      </out>
   </xslt:template>
</xslt:transform>
