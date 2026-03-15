<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">

   <!-- Testcase with @xpath-default-namespace set to the XMLSchema namespace 
        on the parent of a variable that has as="double" attribute.  -->

   <xslt:template match="/">
      <out xslt:xpath-default-namespace="http://www.w3.org/2001/XMLSchema">
         <xslt:variable name="v" select="2.0" as="double"/>
         <xslt:value-of select="$v instance of double"/>
      </out>
   </xslt:template>
</xslt:transform>
