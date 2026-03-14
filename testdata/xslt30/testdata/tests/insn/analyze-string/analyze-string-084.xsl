<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test with regex-group() inside matching-substring with an argument coming from a node.
				Absolute path must be used to reach the node.-->

   <xslt:template match="/doc">
      <out>
         <xslt:analyze-string select="p" regex="(this)">
            <xslt:matching-substring>
               <p>
                  <xslt:value-of select="regex-group(xs:integer(doc('analyze-string-084.xml')//b))"/>
               </p>
            </xslt:matching-substring>
         </xslt:analyze-string>
      </out>
   </xslt:template>
</xslt:transform>
