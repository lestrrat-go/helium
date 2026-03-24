<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
    <!-- Purpose: Test for likely optimizations of descendant patterns to make sure they're done correctly. -->


<xsl:template match="north">
  <out>
  <!-- DS meands the location path is optimizable as a single descendant iterator. -->
DS   1. AC: <xsl:value-of select="name(/descendant-or-self::north)"/>
DS   2. AD: <xsl:value-of select="name(/descendant::near-north)"/>
DS   3. BC: <xsl:value-of select="name(self::node()/descendant-or-self::north)"/>
DS   4. BD: <xsl:value-of select="name(self::node()/descendant::near-north)"/>
NDS  5. CC: <xsl:value-of select="name(descendant-or-self::north/descendant-or-self::north)"/>
NDS  6. CD: <xsl:value-of select="name(descendant-or-self::north/descendant::near-north)"/>
NDS  7. CE: <xsl:value-of select="name(descendant-or-self::north/child::near-north)"/>
NDS  8. DC: <xsl:value-of select="name(descendant::near-north/descendant-or-self::near-north)"/>
NDS  9. DD: <xsl:value-of select="name(descendant::near-north/descendant::far-west)"/>

NDS 10. ACC: <xsl:value-of select="name(/descendant-or-self::north/descendant-or-self::north)"/>
NDS 11. ACE: <xsl:value-of select="name(/descendant-or-self::north/child::near-north)"/>
NDS 12. ADC: <xsl:value-of select="name(/descendant::near-north/descendant-or-self::near-north)"/>
NDS 13. BCC: <xsl:value-of select="name(self::node()/descendant-or-self::north/descendant-or-self::north)"/>
NDS 14. BCE: <xsl:value-of select="name(self::node()/descendant-or-self::north/child::near-north)"/>
NDS 15. BDC: <xsl:value-of select="name(self::node()/descendant::near-north/descendant-or-self::far-west)"/>
NDS 16. BDE: <xsl:value-of select="name(self::node()/descendant::near-north/child::far-west)"/>
NDS 17. CCC: <xsl:value-of select="name(descendant-or-self::north/descendant-or-self::north/descendant-or-self::north)"/>
NDS 18. CCE: <xsl:value-of select="name(descendant-or-self::north/descendant-or-self::north/child::near-north)"/>
NDS 19. CDC: <xsl:value-of select="name(descendant-or-self::north/descendant::near-north/descendant-or-self::near-north)"/>
NDS 20. CDE: <xsl:value-of select="name(descendant-or-self::north/descendant::near-north/child::far-west)"/>
NDS 21. CEC: <xsl:value-of select="name(descendant-or-self::north/child::near-north/descendant-or-self::near-north)"/>
NDS 22. CEE: <xsl:value-of select="name(descendant-or-self::north/child::near-north/child::far-west)"/>
NDS 23. DCC: <xsl:value-of select="name(descendant::near-north/descendant-or-self::near-north/descendant-or-self::near-north)"/>
NDS 24. DCE: <xsl:value-of select="name(descendant::near-north/descendant-or-self::near-north/child::far-west)"/>
NDS 25. DDC: <xsl:value-of select="name(descendant::near-north/descendant::far-west/descendant-or-self::far-west)"/>

DS  26. CC: <xsl:value-of select="name(descendant-or-self::node()/descendant-or-self::north)"/>
DS  27. CD: <xsl:value-of select="name(descendant-or-self::node()/descendant::near-north)"/>
DS  28. CE: <xsl:value-of select="name(descendant-or-self::node()/child::near-north)"/>
DS  29. DC: <xsl:value-of select="name(descendant::node()/descendant-or-self::near-north)"/>
DS  30. DD: <xsl:value-of select="name(descendant::node()/descendant::far-west)"/>

DS  31. ACC: <xsl:value-of select="name(/descendant-or-self::node()/descendant-or-self::north)"/>
DS  32. ACE: <xsl:value-of select="name(/descendant-or-self::node()/child::near-north)"/>
DS  33. ADC: <xsl:value-of select="name(/descendant::node()/descendant-or-self::near-north)"/>
DS  34. BCC: <xsl:value-of select="name(self::node()/descendant-or-self::node()/descendant-or-self::north)"/>
DS  35. BCE: <xsl:value-of select="name(self::node()/descendant-or-self::node()/child::near-north)"/>
DS  36. BDC: <xsl:value-of select="name(self::node()/descendant::node()/descendant-or-self::far-west)"/>
DS  37. BDE: <xsl:value-of select="name(self::node()/descendant::node()/child::far-west)"/>
DS  38. CCC: <xsl:value-of select="name(descendant-or-self::node()/descendant-or-self::node()/descendant-or-self::north)"/>
DS  39. CCE: <xsl:value-of select="name(descendant-or-self::node()/descendant-or-self::node()/child::near-north)"/>
DS  40. CDC: <xsl:value-of select="name(descendant-or-self::node()/descendant::node()/descendant-or-self::near-north)"/>
DS  41. CDE: <xsl:value-of select="name(descendant-or-self::node()/descendant::node()/child::far-west)"/>
DS  42. CEC: <xsl:value-of select="name(descendant-or-self::node()/child::node()/descendant-or-self::near-north)"/>
DS  43. CEE: <xsl:value-of select="name(descendant-or-self::node()/child::node()/child::far-west)"/>
DS  44. DCC: <xsl:value-of select="name(descendant::node()/descendant-or-self::node()/descendant-or-self::near-north)"/>
DS  45. DCE: <xsl:value-of select="name(descendant::node()/descendant-or-self::node()/child::far-west)"/>
DS  46. DDC: <xsl:value-of select="name(descendant::node()/descendant::node()/descendant-or-self::far-west)"/>

  </out>
</xsl:template>

</xsl:stylesheet>