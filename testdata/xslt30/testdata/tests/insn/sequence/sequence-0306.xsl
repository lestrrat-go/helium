<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test case to verify that no error is raised for using xs:untypedAtomic 
  				as the first argument of 'to' operand if the value can be casted to an 
  				integer. -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="/">
      <xslt:variable name="bool" as="xs:untypedAtomic" select="xs:untypedAtomic('5')"/>
      <out>
         <xslt:value-of select="($bool to 5 )"/>
      </out>
   </xslt:template>
</xslt:transform>
