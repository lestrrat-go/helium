<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Use regex-group() inside @group-starting-with of xsl:for-each-group.
     Verify that the set of captured substrings is set to an empty sequence inside a pattern. -->

   <t:variable name="v1">
      <doc>
         <str1>zzzaBcDeFgHiJkLmNoPq</str1>
         <item attr="">string2</item>
         <item attr="a">string1</item>
         <item attr="a">string3</item>
         <item attr="a">string4</item>
         <item attr="">string5</item>
         <item attr="">string6</item>
         <item attr="">string7</item>
      </doc>
   </t:variable>

   <t:template match="doc">
      <out>
         <t:analyze-string select="str1" regex="(a)B(c)D(e)F(g)">
            <t:matching-substring>
               <empty id="match{position()}-1">
                  <t:for-each-group select="$v1//item"
                     group-starting-with="//item[@attr=string(regex-group(6))]">
                     <t:value-of select="."/>
                  </t:for-each-group>
               </empty>
               <empty id="match{position()}-2">
                  <t:for-each-group select="$v1//item"
                     group-starting-with="//item[@attr=string(regex-group(1))]">
                     <t:value-of select="."/>
                  </t:for-each-group>
               </empty>
            </t:matching-substring>
            <t:non-matching-substring>
               <empty id="non-match{position()}">
                  <t:value-of select="."/>
               </empty>
            </t:non-matching-substring>
         </t:analyze-string>
      </out>
   </t:template>
</t:transform>
