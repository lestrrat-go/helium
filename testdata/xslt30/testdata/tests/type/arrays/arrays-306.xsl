<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:tan="tag:textalign.net,2015:ns"
   xmlns:array="http://www.w3.org/2005/xpath-functions/array" expand-text="true" default-mode="test"
   version="3.0">

   <xsl:function name="tan:array-permutations" as="array(*)" visibility="public">
      <!-- Input: any array -->
      <!-- Output: an array whose members are sequences representing the permutations of each item in each member in the 
         input array. -->
      <!-- Example: [(1, 2), 'dog'] becomes [(1, 'dog'), (2, 'dog')] -->
      <!-- The output array will always have a size equal to the product of the item count in each input array member, and the 
         output array's members will share the exact same item count. -->
      <!--kw: arrays -->
      <xsl:param name="input-array" as="array(*)"/>
      <xsl:variable name="array-size" as="xs:integer" select="array:size($input-array)"/>
      <xsl:variable name="array-parts" as="array(*)*">
         <xsl:iterate select="1 to $array-size">
            <xsl:param name="permutations-so-far" as="array(*)*"/>
            <xsl:on-completion select="$permutations-so-far"/>
            <xsl:variable name="this-member-number" as="xs:integer" select="."/>
            <xsl:variable name="array-member" as="item()*"
               select="$input-array($this-member-number)"/>
            <xsl:variable name="new-permutations" as="array(*)+">
               <xsl:choose>
                  <xsl:when test="$this-member-number eq 1">
                     <xsl:sequence select="
                           for $i in $array-member
                           return
                              [$i]"/>
                  </xsl:when>
                  <xsl:otherwise>
                     <xsl:sequence select="
                           for $i in $permutations-so-far,
                              $j in $array-member
                           return
                              [($i(1), $j)]"/>
                  </xsl:otherwise>
               </xsl:choose>
            </xsl:variable>
            <xsl:next-iteration>
               <xsl:with-param name="permutations-so-far" select="$new-permutations"/>
            </xsl:next-iteration>
         </xsl:iterate>
      </xsl:variable>
      <xsl:sequence select="array:join($array-parts)"/>
   </xsl:function>

   <xsl:variable name="test-array" as="array(xs:decimal+)" select="
         [
            (0.0060790273556231, 0.018237082066869301, 0.027355623100303951, 0.045592705167173252, 0.060790273556231003, 0.075987841945288754, 0.094224924012158055, 0.103343465045592705, 0.109422492401215805, 0.133738601823708207, 0.142857142857142857, 0.167173252279635258, 0.176291793313069909, 0.19756838905775076, 0.212765957446808511, 0.227963525835866261, 0.240121580547112462, 0.261398176291793313, 0.273556231003039514, 0.288753799392097264, 0.300911854103343465, 0.328267477203647416, 0.340425531914893617, 0.358662613981762918, 0.367781155015197568, 0.379939209726443769, 0.39513677811550152, 0.40425531914893617, 0.416413373860182371, 0.434650455927051672, 0.443768996960486322, 0.468085106382978723, 0.480243161094224924, 0.501519756838905775, 0.516717325227963526, 0.522796352583586626, 0.553191489361702128, 0.568389057750759878, 0.583586626139817629, 0.60790273556231003, 0.617021276595744681, 0.629179331306990881, 0.644376899696048632, 0.653495440729483283, 0.665653495440729483, 0.686930091185410334, 0.705167173252279635, 0.726443768996960486, 0.738601823708206687, 0.750759878419452888, 0.768996960486322188, 0.80243161094224924, 0.820668693009118541, 0.829787234042553191, 0.848024316109422492, 0.860182370820668693, 0.875379939209726444, 0.893617021276595745, 0.902735562310030395, 0.924012158054711246, 0.936170212765957447, 0.957446808510638298, 0.969604863221884498, 0.99088145896656535),
            (0.0060790273556231, 0.027355623100303951, 0.045592705167173252, 0.060790273556231003, 0.075987841945288754, 0.103343465045592705, 0.109422492401215805, 0.133738601823708207, 0.142857142857142857, 0.176291793313069909, 0.227963525835866261, 0.240121580547112462, 0.273556231003039514, 0.288753799392097264, 0.300911854103343465, 0.328267477203647416, 0.340425531914893617, 0.367781155015197568, 0.379939209726443769, 0.39513677811550152, 0.40425531914893617, 0.416413373860182371, 0.443768996960486322, 0.480243161094224924, 0.516717325227963526, 0.522796352583586626, 0.568389057750759878, 0.583586626139817629, 0.60790273556231003, 0.629179331306990881, 0.644376899696048632, 0.653495440729483283, 0.665653495440729483, 0.686930091185410334, 0.705167173252279635, 0.726443768996960486, 0.738601823708206687, 0.750759878419452888, 0.787234042553191489, 0.80243161094224924, 0.829787234042553191, 0.848024316109422492, 0.860182370820668693, 0.875379939209726444, 0.902735562310030395, 0.924012158054711246, 0.969604863221884498, 0.99088145896656535),
            (0.378378378378378378, 0.648648648648648649, 0.756756756756756757, 0.891891891891891892)
         ]"> </xsl:variable>

   <xsl:variable name="test-array-permuted" as="array(xs:decimal+)"
      select="tan:array-permutations($test-array)"/>

   <xsl:output indent="yes"/>
   <xsl:template name="xsl:initial-template">
      <output>
         <xsl:value-of select="array:size($test-array-permuted)"/>
      </output>
   </xsl:template>


</xsl:stylesheet>
