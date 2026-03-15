<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>

    
    <!-- Test of xsl:source-document with avg() -->
    
    <xsl:template name="s-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="round(avg(./BOOKLIST/BOOKS/ITEM/PRICE), 2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with avg(), filtered with a motionless predicate -->
    
    <xsl:template name="s-004" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="avg(./BOOKLIST/BOOKS/ITEM[@CAT='P']/PRICE)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with avg(), filtered with a positional predicate -->
    
    <xsl:template name="s-005" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="round(avg(./BOOKLIST/BOOKS/ITEM[position() lt 4]/PRICE), 2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with avg(), filtered with a positional predicate -->
    
    <xsl:template name="s-006" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:copy-of select="round(avg(./BOOKLIST/BOOKS/*[position() lt 4]/PRICE), 2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with avg(), filtered with a positional predicate -->
    
    <xsl:template name="s-007" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:sequence select="round(avg(./BOOKLIST/BOOKS/*:ITEM[position() lt 4]/PRICE), 2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with avg(), filtered using data() to make it streamable -->
    
    <xsl:template name="s-008" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(avg(./BOOKLIST/BOOKS/ITEM/PAGES/data()[. &lt; 1000][. &gt; 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with avg(), filtered using text() to make it streamable -->
    
    <xsl:template name="s-009" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(avg(./BOOKLIST/BOOKS/ITEM/PAGES/text()[. &lt; 1000][. &gt; 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with avg(), filtered using number() to make it streamable -->
    
    <xsl:template name="s-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(avg(./BOOKLIST/BOOKS/ITEM/PAGES/number()[. &lt; 1000][. &gt; 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with avg(), selecting nothing and returning empty result -->
    
    <xsl:template name="s-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="avg(ITEM/PAGES)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of avg() selecting both streamed nodes and literals -->
    
    <xsl:template name="s-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="avg((./BOOKLIST/BOOKS/ITEM/PAGES/number(), 31, 32))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of avg() selecting both streamed nodes and literals while also filtering -->
    
    <xsl:template name="s-013" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(avg((tail(./BOOKLIST/BOOKS/ITEM/PAGES)/number(), 31, 32)), 2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of avg() output feeding directly into xsl:source-document -->
    
    <xsl:template name="s-014" use-when="$RUN">
      <out>
        <xsl:source-document streamable="yes" href="../docs/books.xml">
          <xsl:value-of select="round(avg(//QUANTITY/number()), 2)"/>
        </xsl:source-document>
      </out>  
    </xsl:template>
       
    
    <!-- Test of xsl:source-document with sum of a computed value -->
    
    <xsl:template name="s-015" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="format-number(avg(for $d in data(outermost(//DIMENSIONS))
                                                  return let $x := tokenize($d, '\s')!number() 
                                                  return $x[1]*$x[2]*$x[3]), '99.999')"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with sum of a computed value -->
    
    <xsl:template name="s-016" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="xs:integer(round(avg(account/transaction/(@value*2))))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with avg(), filtered using copy-of() to make it streamable -->
    
    <xsl:template name="s-017" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="round(avg(./BOOKLIST/BOOKS/ITEM/PAGES/copy-of()[. &lt; 1000][. &gt; 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with avg() applied to an attribute -->
    
    <xsl:template name="s-018" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(avg(account/transaction/@value))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with avg() over attribute values, filtered -->
    
    <xsl:template name="s-019" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(avg(account/transaction/@value[xs:decimal(.) gt 0]))"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- Test of xsl:source-document with avg() over attribute values, computed -->
    
    <xsl:template name="s-020" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(avg(account/transaction/abs(@value)))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- avg() of a conditional value -->
    
    <xsl:template name="s-021" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="round(avg(account/transaction/
                (if (xs:date(@date) lt xs:date('2020-01-01')) then +@value else (@value+1))))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- avg() with a loop used to compute each item -->
    
    <xsl:template name="s-022" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="avg(BOOKLIST/BOOKS/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document calling avg() using striding select expr with grounding mapping operator of the form /x/y!z -->
    
    <xsl:template name="s-023" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="avg(/BOOKLIST/BOOKS/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming avg(): grounded operand, selects nothing -->
    
    <xsl:template name="s-040" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="avg(data(/BOOKLIST/BOOKS/NOTHING))" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming avg(): striding operand, selects nothing -->
    
    <xsl:template name="s-041" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="avg(/BOOKLIST/BOOKS/MAGAZINE)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming avg(): rooted striding operand, selects nothing -->
    
    <xsl:template name="s-042" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="avg(/BOOKLIST/BOOKS/MAGAZINE)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming avg(): climbing operand, selects nothing -->
    
    <xsl:template name="s-043" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="avg(//PRICE/../@nothing)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming avg(): dayTimeDuration values -->
    
    <xsl:template name="s-050" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="avg(/BOOKLIST/BOOKS/ITEM/PUB-DATE ! (xs:date('2050-01-01') - xs:date(.)))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming avg(): yearMonthDuration values -->
    
    <xsl:template name="s-051" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="avg(
                                    for $pubdate in /BOOKLIST/BOOKS/ITEM/PUB-DATE/xs:date(.) return
                                    let $monthsAgo := (2050*12) - (12*year-from-date($pubdate) + month-from-date($pubdate)) return
                                    xs:yearMonthDuration('P1M') * $monthsAgo)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming avg(): inconsistent type values -->
    
    <xsl:template name="s-052" use-when="$RUN">
      <xsl:param name="p" select="23"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="avg(($p, /BOOKLIST/BOOKS/ITEM/PUB-DATE ! (xs:date('2050-01-01') - xs:date(.))))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming avg(): incorrect type values -->
    
    <xsl:template name="s-053" use-when="$RUN">
      <xsl:param name="p" select="'Wrong'"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="avg(($p, /BOOKLIST/BOOKS/ITEM/PUB-DATUM ! (xs:date('2050-01-01') - xs:date(.))))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming avg(): inconsistent type values, error is caught -->
    
    <xsl:template name="s-054" use-when="$RUN">
      <xsl:param name="p" select="23"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try>
            <xsl:value-of select="avg(($p, /BOOKLIST/BOOKS/ITEM/PUB-DATE ! (xs:date('2050-01-01') - xs:date(.))))"/>
            <xsl:catch errors="err:FORG0006" select="'caught'"/>
          </xsl:try>  
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming avg(): incorrect type values, error is caught -->
    
    <xsl:template name="s-055" use-when="$RUN">
      <xsl:param name="p" select="'Wrong'"/>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:try>
            <xsl:value-of select="avg(($p, /BOOKLIST/BOOKS/ITEM/PUB-DATUM ! (xs:date('2050-01-01') - xs:date(.))))"/>
            <xsl:catch errors="err:FORG0006" select="'caught'"/>
          </xsl:try>
        </out>
      </xsl:source-document>
    </xsl:template>
                                               
    
</xsl:stylesheet>