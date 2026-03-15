<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:t="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="3.0">
<!-- Purpose: Test collection() function within a package, overridden by another package
      with different xsl:strip-space declarations. -->
   
   <t:strip-space elements=""/>
   
   
   <t:use-package name="http://www.w3.org/xslts/collection-006">
      <t:override>
         <t:template name="a">
            <pack-b>
               <t:copy-of select="collection('collection-006.cxn')"/>
            </pack-b>
            <t:call-template name="t:original"/>           
         </t:template>
      </t:override>
   </t:use-package>
   
   
   <t:template name="t:initial-template">
      <out>
         <t:call-template name="a"/>
      </out>
   </t:template>
   
 
</t:transform>
