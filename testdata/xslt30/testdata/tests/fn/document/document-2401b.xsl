<?xml version="1.0" encoding="UTF-8"?>
<t:package xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" name="http://www.w3.org/xslts/document-2402"
   package-version="3.0" exclude-result-prefixes="xs" version="3.0">
   <!-- Purpose: Test document() function within a package, overriding by another package
      with different xsl:strip-space declarations. -->



   <t:strip-space elements="*"/>

   <t:use-package name="http://www.w3.org/xslts/document-2401a" package-version="3.0">

      <t:override>
         <t:template name="a" visibility="public">
            <out>
               <stripped>
                  <t:value-of select="count(document('doc14.xml')/*/text())"/>
               </stripped>
               <t:call-template name="t:original"/> 
            </out>
         </t:template>
      </t:override>

   </t:use-package>

</t:package>
