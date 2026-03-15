<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test that system-property can be dynamically evaluated via function-lookup.-->
   
   <xslt:param name="arity" select="1"/>
   <xslt:template match="/" xmlns:fun="http://www.w3.org/2005/xpath-functions">
      <xslt:variable name="expr" select="function-lookup(xs:QName('fun:system-property'), $arity)"/>
      <out>
         <xslt:value-of select="$expr('xslt:version') castable as xs:decimal"/>
      </out>
   </xslt:template>
</xslt:transform>
